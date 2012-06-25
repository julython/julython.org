from django.conf.urls.defaults import patterns, url

urlpatterns = patterns('july.people.views',
    url(r'^(?P<username>[a-zA-Z0-9_]+)$', 'user_profile', name='member-profile'),
    url(r'^(?P<username>[a-zA-Z0-9_]+)/edit/$', 'edit_profile', name='edit-profile'),
    url(r'^(?P<username>[a-zA-Z0-9_]+)/email/(?P<email>.*)$', 'delete_email', name='delete-email'),
)
