
import unittest
import datetime

# This needs to be first to setup imports
from utils import WebTestCase

from google.appengine.ext import ndb

from july.people.models import Accumulator

class TestAccumulator(WebTestCase):
    
    def test_make_key(self):
        key = Accumulator.make_key('test', 7)
        self.assertEqual(key.id(), 'test:7')
    
    def test_get_histogram(self):
        acc1 = Accumulator.get_or_insert('test:7')
        acc2 = Accumulator.get_or_insert('test:8')
        acc1.count = 4
        acc2.count = 6
        ndb.put_multi([acc1, acc2])
        counts = Accumulator.get_histogram('test')
        self.assertListEqual(counts, 
            [0,0,0,0,0,0,4,6,0,0,
             0,0,0,0,0,0,0,0,0,0,
             0,0,0,0,0,0,0,0,0,0,0])
    
    def test_add_count(self):
        t = datetime.datetime(year=2012, month=7, day=2, hour=0, minute=5)
        Accumulator.add_count('test', t, 5)
        Accumulator.add_count('test', t + datetime.timedelta(days=2), 7)
        Accumulator.add_count('test', t + datetime.timedelta(days=9))
        
        counts = Accumulator.get_histogram('test')
        self.assertListEqual(counts, 
            [0,5,0,7,0,0,0,0,0,0,
             1,0,0,0,0,0,0,0,0,0,
             0,0,0,0,0,0,0,0,0,0,0])
        


if __name__ == "__main__":
    unittest.main()