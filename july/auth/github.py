import logging

from social_auth.backends import USERNAME
from social_auth.backends.contrib import github
from july.people.models import Location

class GithubBackend(github.GithubBackend):
    
    def get_user_details(self, response):
        """Return user details from Github account"""
        data = {
            USERNAME: response.get('login'),
            'email': response.get('email') or '',
            'fullname': response['name'],
            'last_name': '',
            'url': response.get('blog', ''),
            'description': response.get('bio', ''),
            'picture_url': response.get('avatar_url', '')        
        }
        
        try:
            data['first_name'], data['last_name'] = response['name'].split(' ', 1)
        except:
            data['first_name'] = response.get('name') or 'Annon'
            
        try:
            location = response.get('location', '')
            if location:
                data['location'], _ = Location.create(location)
        except:
            logging.exception('Problem finding location') 
        
        return data

# Backend definition
BACKENDS = {
    'github': github.GithubAuth,
}
