__author__ = 'Kevin'
from social_auth.backends.twitter import TwitterAuth, TwitterBackend as BaseTwitterBackend

class TwitterBackend(BaseTwitterBackend):
    """Twitter OAuth authentication backend"""

    def extra_data(self, user, uid, response, details):
        d = super(TwitterBackend, self).extra_data(user, uid, response, details)
        d['profile_image_url_https'] = response['profile_image_url_https']
        d['screen_name'] = response['screen_name']
        #if not user.picture_url:
        #TODO - Consider whether it's a bad idea to save user data here.
        if not user.picture_url:
            user.picture_url = 'https://api.twitter.com/1/users/profile_image/{screen_name}?size=bigger'.format(
                                                                                screen_name=response.get('screen_name'))
            user.save()
        return d


# Backend definition
BACKENDS = {
    'twitter': TwitterAuth,
    }
