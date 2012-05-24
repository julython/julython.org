"""
Cache page anonymous users only.
http://djangosnippets.org/snippets/2230/
"""

from django.utils.decorators import decorator_from_middleware_with_args
from django.middleware.cache import CacheMiddleware


# Note: The standard Django decorator cache_page doesn't give you the anon-nonanon flexibility, 
# and the standard Django 'full-site' cache middleware forces you to cache all pages. 
def cache_page_anonymous(*args, **kwargs):
    """
    Decorator to cache Django views only for anonymous users.
    Use just like the decorator cache_page:

    @cache_page_anonymous(60 * 30)  # cache for 30 mins
    def your_view_here(request):
        ...
    """
    key_prefix = kwargs.pop('key_prefix', None)
    return decorator_from_middleware_with_args(CacheMiddleware)(cache_timeout=args[0], 
                                                                key_prefix=key_prefix, 
                                                                cache_anonymous_only=True)