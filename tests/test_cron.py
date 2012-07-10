
import unittest
import logging
import datetime
import json
import base64

from google.appengine.ext import ndb, deferred

from utils import WebTestCase

from july.cron import app, fix_orphans


class CronTests(WebTestCase):
    
    APPLICATION = app
    
    def test_fix_commits(self):
        self.make_user('email:sam@email.com')
        self.make_orphan('sam@email.com', "Foo", hash='1234')
        self.make_orphan('sam@email.com', "Foo", hash='12345')
        self.make_orphan('tam@email.com', "Foo", hash='12346')
        self.make_orphan('cam@email.com', "Foo", hash='7869')
        fix_orphans(email='sam@email.com')
        
        # check the task queue
        tasks = self.taskqueue_stub.GetTasks('default')
        self.assertEqual(2, len(tasks))

        # Run the task
        task = tasks[0]
        deferred.run(base64.b64decode(task['body']))

if __name__ == '__main__':
    logging.getLogger().setLevel(logging.DEBUG)
    unittest.main()