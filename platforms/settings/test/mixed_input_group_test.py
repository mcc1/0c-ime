"""Real Chromium regression tests: python3 platforms/settings/test/mixed_input_group_test.py.

Requires Playwright and Chromium; mocks only the Tauri command boundary, using
the actual HTML/CSS/JS. Does not run an engine or access personal settings.
"""
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import shutil
import threading
import unittest

# Options that only matter while mixed input is on, in display order.
CHILDREN = [("mixed-input-policy", "混打組字傾向", ["conservative", "aggressive"], "aggressive"),
            ("neighbor-key-recovery", "鄰鍵容錯", ["off", "on"], "on")]


class QuietHandler(SimpleHTTPRequestHandler):
    def log_message(self, *_):
        pass


class MixedInputGroupTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        try:
            from playwright.sync_api import sync_playwright
        except ImportError:
            raise unittest.SkipTest("requires Playwright")
        executable = shutil.which(os.environ.get("CHROMIUM", "chromium"))
        if not executable:
            raise unittest.SkipTest("requires Chromium (or CHROMIUM)")
        cls.playwright = sync_playwright().start()
        cls.browser = cls.playwright.chromium.launch(executable_path=executable, headless=True, args=["--no-sandbox"])
        directory = Path(__file__).resolve().parents[1] / "ui"
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), partial(QuietHandler, directory=str(directory)))
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.thread.join()
        cls.server.server_close()
        cls.browser.close()
        cls.playwright.stop()

    def open(self, items):
        fixture = {"items": items, "file": {"state": "ok", "read_only": False}, "features": {"vocab": False}}
        page = self.browser.new_page(viewport={"width": 960, "height": 680})
        self.addCleanup(page.close)
        errors = []
        page.on("pageerror", lambda error: errors.append(str(error)))
        page.add_init_script("""
          window.setInterval = () => 0;
          window.__TAURI__ = {
            core: {invoke: async (command) => {
              if (command === 'app_info') return {platform:'windows/x86_64'};
              if (command === 'get_settings') return FIXTURE;
              if (command === 'get_status') return {schema:1, installed:'0.2.8', mode:'manual', runtime:{state:'not_running'}, update:{state:'never_checked'}};
              throw new Error('unexpected command: ' + command);
            }}, event: {listen: async () => () => {}}
          };
        """.replace("FIXTURE", json.dumps(fixture, ensure_ascii=False)))
        page.goto(f"http://127.0.0.1:{self.server.server_port}/index.html?page=mode")
        page.locator('[data-focus="mixed-input"]').wait_for(state="attached")
        return page, errors

    def test_options_are_children_of_mixed_input(self):
        items = [{"key": "mixed-input", "label": "中英混打", "values": ["off", "on"], "value": "on"}]
        items += [{"key": k, "label": label, "values": values, "value": value} for k, label, values, value in CHILDREN]
        page, errors = self.open(items)
        group = page.locator('[role="group"][aria-labelledby="label-mixed-input"]')
        self.assertEqual(group.count(), 1)
        self.assertEqual(group.locator('.suboption').count(), len(CHILDREN))
        self.assertEqual(group.locator('.label').all_text_contents(), ['中英混打'] + [label for _, label, _, _ in CHILDREN])
        for key, _, _, _ in CHILDREN:
            self.assertEqual(page.locator(f'#page-candidates [data-focus="{key}"]').count(), 0)
        self.assertEqual(errors, [])

    def test_children_follow_parent_regardless_of_item_order(self):
        items = [{"key": k, "label": label, "values": values, "value": value} for k, label, values, value in CHILDREN]
        items.append({"key": "mixed-input", "label": "中英混打", "values": ["off", "on"], "value": "off"})
        page, errors = self.open(items)
        self.assertEqual(page.locator('[role="group"][aria-labelledby="label-mixed-input"] .suboption').count(), len(CHILDREN))
        self.assertEqual(errors, [])


if __name__ == '__main__':
    unittest.main()
