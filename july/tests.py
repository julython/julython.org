import datetime
import mock

from django.test import TestCase


class JulyViews(TestCase):

    def test_index(self):
        resp = self.client.get('/')
        self.assertEqual(resp.status_code, 200)

    def test_help(self):
        resp = self.client.get('/help/')
        self.assertEqual(resp.status_code, 200)

    def test_live(self):
        resp = self.client.get('/live/')
        self.assertEqual(resp.status_code, 200)

    def test_register_get(self):
        resp = self.client.get('/register/')
        self.assertEqual(resp.status_code, 200)

    def test_register_bad(self):
        resp = self.client.post('/register/', {'Bad': 'field'})
        self.assertEqual(resp.status_code, 200)
        self.assertContains(resp, "This field is required.")

    def test_register_good(self):
        post = {
            'username': 'fred',
            'password1': 'secret',
            'password2': 'secret'
        }
        resp = self.client.post('/register/', post)
        self.assertRedirects(resp, '/')


class AbuseTests(TestCase):

    def test_set_abuse(self):
        from django.conf import settings
        settings.ABUSE_LIMIT = 3  # 3 times !

        from middleware import AbuseMiddleware
        today = datetime.date.today()
        request = mock.MagicMock()
        request.session = {}
        mid = AbuseMiddleware()

        abuse_reported = mid._abuse_reported(request)
        can_report_abuse = mid._can_report_abuse(request)

        abuse_reported()  # one
        self.assertEqual(
            request.session['abuse_date'],
            today - datetime.timedelta(days=2),
        )
        self.assertTrue(can_report_abuse())

        abuse_reported()  # two
        self.assertEqual(
            request.session['abuse_date'],
            today - datetime.timedelta(days=1),
        )
        self.assertTrue(can_report_abuse())

        abuse_reported()  # tree
        self.assertEqual(
            request.session['abuse_date'],
            today,
        )
        self.assertFalse(can_report_abuse())  # game is over !

    def test_reset_abuse(self):
        from django.conf import settings
        settings.ABUSE_LIMIT = 3

        from middleware import AbuseMiddleware
        today = datetime.date.today()
        request = mock.MagicMock()
        request.session = {'abuse_date': today-datetime.timedelta(days=10)}
        mid = AbuseMiddleware()

        abuse_reported = mid._abuse_reported(request)
        can_report_abuse = mid._can_report_abuse(request)

        abuse_reported()  # if abuse_date is old enugh it should be reseted
        self.assertEqual(
            request.session['abuse_date'],
            today - datetime.timedelta(days=2),
        )
        self.assertTrue(can_report_abuse())

        request.session = {'abuse_date': today-datetime.timedelta(days=3)}

        abuse_reported()
        self.assertEqual(
            request.session['abuse_date'],
            today - datetime.timedelta(days=2),
        )
        self.assertTrue(can_report_abuse())

        request.session = {'abuse_date': today-datetime.timedelta(days=2)}

        abuse_reported()
        self.assertEqual(
            request.session['abuse_date'],
            today - datetime.timedelta(days=1),
        )
        self.assertTrue(can_report_abuse())

        abuse_reported()
        self.assertEqual(
            request.session['abuse_date'],
            today,
        )
        self.assertFalse(can_report_abuse())
