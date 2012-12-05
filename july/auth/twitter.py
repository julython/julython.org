__author__ = 'Kevin'
from social_auth.backends.twitter import TwitterAuth, TwitterBackend as BaseTwitterBackend

class TwitterBackend(BaseTwitterBackend):
    """Twitter OAuth authentication backend"""

    def extra_data(self, user, uid, response, details):
        d = super(TwitterBackend, self).extra_data(user, uid, response, details)
        d['profile_image_url_https'] = response['profile_image_url_https']
        print response
        if not user.picture_url:
            #TODO - Consider whether it's a bad idea to save user data here.
            user.picture_url = response.get('profile_image_url_https','')
            user.save()
        return d


# Backend definition
BACKENDS = {
    'twitter': TwitterAuth,
    }
