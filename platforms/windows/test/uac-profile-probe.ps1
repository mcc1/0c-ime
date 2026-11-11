# Read-only UAC transition probe. No keys, window titles, document text or settings changes.
param([ValidateRange(10,300)][int]$Seconds=60,
      [string]$OutputPath=(Join-Path ([IO.Path]::GetTempPath()) ('0c-uac-'+[Guid]::NewGuid().ToString('N')+'.json')))
$ErrorActionPreference='Stop'
if (-not ('ZcUacProbe' -as [type])) {
Add-Type @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class ZcUacProbe {
 [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
 [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr window, out uint process);
 [DllImport("user32.dll")] public static extern IntPtr GetKeyboardLayout(uint thread);
 [DllImport("user32.dll",SetLastError=true)] static extern IntPtr OpenInputDesktop(uint flags, bool inherit, uint access);
 [DllImport("user32.dll",CharSet=CharSet.Unicode,SetLastError=true)] static extern bool GetUserObjectInformation(IntPtr handle,int index,StringBuilder buffer,uint length,out uint needed);
 [DllImport("user32.dll")] static extern bool CloseDesktop(IntPtr handle);
 public static string InputDesktop() {
  IntPtr h=OpenInputDesktop(0,false,1);
  if(h==IntPtr.Zero) return "unavailable:"+Marshal.GetLastWin32Error();
  try {
   var b=new StringBuilder(256);uint n;
   return GetUserObjectInformation(h,2,b,512,out n)?b.ToString():"unavailable:"+Marshal.GetLastWin32Error();
  } finally {CloseDesktop(h);}
 }
}
'@
}
$rows=New-Object 'System.Collections.Generic.List[object]'
$previous=''
$watch=[Diagnostics.Stopwatch]::StartNew()
Write-Host "Recording for $Seconds seconds. Focus your normal text window, select 0c, then manually trigger and cancel UAC."
Write-Host "Output: $OutputPath"
try {
 while ($watch.Elapsed.TotalSeconds -lt $Seconds) {
  $window=[ZcUacProbe]::GetForegroundWindow()
  [uint32]$foregroundProcess=0
  $thread=if($window -ne [IntPtr]::Zero){[ZcUacProbe]::GetWindowThreadProcessId($window,[ref]$foregroundProcess)}else{0}
  $layout=if($thread -ne 0){'0x'+[ZcUacProbe]::GetKeyboardLayout($thread).ToInt64().ToString('X16')}else{$null}
  $name=$null
  if($foregroundProcess -ne 0){try{$name=[Diagnostics.Process]::GetProcessById([int]$foregroundProcess).ProcessName}catch{}}
  $workers=@(Get-Process -Name '0c-server','0c-ui','0c-settings' -ErrorAction SilentlyContinue |
    Sort-Object ProcessName,Id | ForEach-Object {@{name=$_.ProcessName;pid=$_.Id}})
  $state=[ordered]@{desktop=[ZcUacProbe]::InputDesktop();foreground_process=$name;foreground_pid=$foregroundProcess;foreground_thread=$thread;foreground_hkl=$layout;workers=$workers}
  $encoded=ConvertTo-Json -InputObject $state -Depth 5 -Compress
  if($encoded -ne $previous){
   $rows.Add([ordered]@{utc=[DateTime]::UtcNow.ToString('o');state=$state})
   $previous=$encoded
  }
  Start-Sleep -Milliseconds 200
 }
} finally {
 $report=[ordered]@{schema=1;duration_seconds=$watch.Elapsed.TotalSeconds;note='HKL is the foreground thread keyboard layout, not a TSF profile GUID. Desktop access failure during UAC is recorded, not bypassed. No input contents were captured.';samples=@($rows.ToArray())}
 ConvertTo-Json -InputObject $report -Depth 8 | Set-Content -LiteralPath $OutputPath -Encoding UTF8
 Write-Host "Saved: $OutputPath"
}
