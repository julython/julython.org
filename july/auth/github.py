import logging

from social_auth.backends.contrib import github


class GithubBackend(github.GithubBackend):
    ID_KEY = 'login'

    def get_user_details(self, response):
        """Return user details from Github account"""
        data = {
            'username': response.get('login'),
            'email': response.get('email') or '',
            'fullname': response.get('name', 'Secret Agent'),
            'last_name': '',
            'url': response.get('blog', ''),
            'description': response.get('bio', ''),
            'picture_url': response.get('avatar_url', '')
        }

        try:
            names = data['fullname'].split(' ')
            data['first_name'], data['last_name'] = names[0], names[-1]
        except:
            data['first_name'] = data['fullname']

        logging.debug("Github Auth: %s", data)
        return data


# Backend definition
BACKENDS = {
    'github': github.GithubAuth,
}
