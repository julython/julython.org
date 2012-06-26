from django.conf.urls.defaults import patterns, url

urlpatterns = patterns('july.people.views',
    url(r'^location/$', 'locations', name='edit-profile'),
    url(r'^location/(?P<location_slug>[a-zA-Z0-9\-]+)/$', 'users_by_location', name='edit-profile'),
    url(r'^(?P<username>[a-zA-Z0-9_]+)$', 'user_profile', name='member-profile'),
    url(r'^(?P<username>[a-zA-Z0-9_]+)/edit/$', 'edit_profile', name='edit-profile'),
    url(r'^(?P<username>[a-zA-Z0-9_]+)/email/(?P<email>.*)$', 'delete_email', name='delete-email'),
)
