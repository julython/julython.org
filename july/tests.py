
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
