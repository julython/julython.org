from django.conf.urls.defaults import patterns, include, url
from gae_django import admin

admin.autodiscover()

urlpatterns = patterns('',
    # Examples:
    url(r'^admin/', include(admin.site.urls)),
    url(r'^$', 'july.views.index', name='index'),
)
