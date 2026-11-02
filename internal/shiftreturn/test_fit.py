#!/usr/bin/env python3
"""Numerical/weighting contracts using hypothetical observations only."""
import math
import unittest
import numpy as np
import fit
from exchange import Insufficient, encoded, parse


def observation(intent,component,event,id,gold,x=0.,fixture=None):
    features=[0.]*13;features[5]=x
    return {'_intent':intent,'component_id':component,'fixture_id':fixture or component,
            'event_index':event,'raw':'hypothetical','gold_ids':gold,
            'candidate':{'id':id,'identity':id,'features':features}}

class WeightContracts(unittest.TestCase):
    def prepare(self,observations):
        observations.sort(key=fit.observation_order)
        means,stds,counts=fit.population(observations)
        pairs,diagnostics=fit.build_pairs(observations,means,stds)
        return means,stds,counts,pairs,diagnostics

    def test_multiple_gold_are_never_negatives(self):
        obs=[observation('return','a',1,id,['g1','g2'],x) for id,x in [('g1',0.),('g2',2.),('negative',1.)]]
        _,_,_,pairs,_=self.prepare(obs)
        self.assertEqual(len(pairs),2)
        self.assertEqual([q for _,q in pairs],[.5,.5])
        self.assertAlmostEqual(sum(q for _,q in pairs),1.)

    def test_no_pair_population_included_weights_redistributed(self):
        obs=[observation('literal','none',1,'literal',['literal'],10.)]
        obs += [observation('return','active',1,id,['gold'],x) for id,x in [('gold',0.),('negative',2.)]]
        means,_,counts,pairs,coverage=self.prepare(obs)
        self.assertEqual(counts[5],3)
        self.assertEqual(means[5],4.)
        self.assertEqual(len(pairs),1)
        self.assertEqual(pairs[0][1],1.)
        self.assertEqual(coverage['no_pair_checkpoints'],1)
        self.assertEqual(coverage['no_pair_rows'],1)
        self.assertEqual(coverage['effective_intents'],['return'])

    def test_hierarchical_component_checkpoint_weights(self):
        obs=[]
        for intent,component,events in [('literal','c',[1]),('return','a',[1,2]),('return','b',[1])]:
            for event in events:
                obs.extend(observation(intent,component,event,id,['gold'],x) for id,x in [('gold',2.),('negative',0.)])
        *_,pairs,_=self.prepare(obs)
        self.assertEqual([q for _,q in pairs],[.5,.125,.125,.25])
        self.assertEqual(sum(q for _,q in pairs),1.)

    def test_missing_columns_excluded_and_constant_columns_zero(self):
        a=observation('return','a',1,'a',['a'],3.);a['candidate']['features'][6]=1
        b=observation('return','a',1,'b',['a'],7.)
        means,stds,counts=fit.population([a,b])
        self.assertEqual((means[5],stds[5],counts[5]),(7.,0.,1))
        self.assertEqual(fit.standardized(a['candidate']['features'],means,stds)[5],0.)
        self.assertEqual(fit.standardized(b['candidate']['features'],means,stds)[5],0.)

    def test_empty_pairs_and_full_budget_fail_closed(self):
        obs=[observation('literal','a',1,'literal',['literal'])]
        with self.assertRaises(Insufficient):self.prepare(obs)
        gold=[str(i) for i in range(1500)]
        obs=[observation('return','a',1,str(i),gold) for i in range(3000)]
        with self.assertRaises(Insufficient):self.prepare(obs)

    def test_fixed_masks_and_simultaneous_reproducibility(self):
        d=np.ones(13);pairs=[(d,1.)]
        first=fit.optimize(pairs,fit.MASKS['E0'])
        second=fit.optimize(pairs,fit.MASKS['E0'])
        self.assertEqual(encoded(first),encoded(second))
        self.assertTrue(all(x==0. for x in first[0][7:]))
        self.assertTrue(all(math.isfinite(x) for x in first[0]))

    def test_population_uses_plain_sequential_float64_addition(self):
        obs=[observation('return','a',1,id,['gold'],x,fixture=id) for id,x in [('a',1e16),('b',1.),('c',-1e16)]]
        obs.sort(key=fit.observation_order)
        means,_,counts=fit.population(obs)
        self.assertEqual(counts[5],3)
        self.assertEqual(means[5],0.)  # CPython3.12 compensated sum would give1/3.

    def test_nonfinite_json_numbers_rejected(self):
        for value in ('NaN','Infinity','1e999'):
            with self.assertRaises(ValueError):parse('{"x":'+value+'}')

if __name__=='__main__':unittest.main()
