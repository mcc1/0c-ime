#!/usr/bin/env python3
"""Reconstruct frozen train design matrix and inspect one already-fitted model.
No coefficient updates, extra estimators, dev labels, or test labels are used.
"""
import sys
import ast
import json
import pathlib
import types

ROOT=pathlib.Path(sys.argv[1]).resolve();BASE=pathlib.Path(sys.argv[2]).resolve()
source=(ROOT/'internal/mixedeval/fit.py').read_text();tree=ast.parse(source)
# Reuse exactly the frozen data construction, stopping before the optimizer.
train=next(n for n in tree.body if isinstance(n,ast.FunctionDef) and n.name=='train')
prefix=[]
for statement in train.body:
    if isinstance(statement,ast.For) and isinstance(statement.target,ast.Name) and statement.target.id=='iteration':break
    prefix.append(statement)
train.body=prefix+[ast.Return(ast.Tuple(elts=[ast.Name('x',ast.Load()),ast.Name('q',ast.Load())],ctx=ast.Load()))]
train.name='design_only';functions=[n for n in tree.body if not (isinstance(n,ast.FunctionDef) and n.name=='select') and not isinstance(n,ast.If)]
ns={'__name__':'frozen_training_diagnostic'};exec(compile(ast.fix_missing_locations(ast.Module(body=functions,type_ignores=[])),'frozen-train-design','exec'),ns)
x,q=ns['design_only'](ROOT,BASE/'enumerate-v1',BASE/'unused-diagnostic-output')
np=ns['np'];w=np.asarray(json.loads((BASE/'trained-v1/model.json').read_text())['coefficients']);margin=x@w;grad=-(x.T@(q*np.exp(-np.logaddexp(0,margin))))+.001*w;grad[12]=0
r={'updates_performed':0,'estimator':'single frozen trained-v1/model.json','pairs':len(x),'weights_sum':float(q.sum()),'zero_objective':float(q.sum()*np.log(2)),'frozen_model_objective':float(q@np.logaddexp(0,-margin)+.0005*(w@w)),'gradient_l2_norm':float(np.linalg.norm(grad)),'max_abs_gradient':float(np.max(np.abs(grad))),'pair_accuracy_unweighted':float(np.mean(margin>0)),'pair_accuracy_balanced_weighted':float(q@(margin>0)/q.sum()),'coefficient_l2_norm':float(np.linalg.norm(w)),'interpretation':'fixed-budget estimator diagnostic, not a new acceptance gate or proof of optimizer convergence'}
(BASE/'training-diagnostic.json').write_text(json.dumps(r,sort_keys=True,indent=2)+'\n');print(json.dumps(r))
