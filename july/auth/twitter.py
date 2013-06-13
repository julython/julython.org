__author__ = 'Kevin'

import logging

from social_auth.backends import twitter
from july.people.models import Location

class TwitterBackend(twitter.TwitterBackend):
    """Twitter OAuth authentication backend"""

    def get_user_details(self, response):
        """Return user details from Twitter account"""
        data = {
            'username': response['screen_name'],
            'email': '',  # not supplied
            'fullname': response['name'],
            'last_name': '',
            'url': response.get('url', ''),
            'description': response.get('description', ''),
            'picture_url': response.get('profile_image_url', '')        
        }
        try:
            data['first_name'], data['last_name'] = response['name'].split(' ', 1)
        except:
            data['first_name'] = response['name']
        try:
            location = response.get('location', '')
            if location:
                data['location'], _ = Location.create(location)
        except:
            logging.exception('Problem finding location') 
        return data

# Backend definition
BACKENDS = {
    'twitter': twitter.TwitterAuth,
    }
