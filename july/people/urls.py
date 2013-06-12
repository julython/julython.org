from django.conf.urls import patterns, url

from july.people import views

urlpatterns = patterns('july.people.views',
    url(r'^(?P<username>[a-zA-Z0-9_]+)/$', 
        views.UserProfile.as_view(), 
        name='member-profile'),
    url(r'^(?P<username>[a-zA-Z0-9_]+)/edit/$', 'edit_profile', name='edit-profile'),
    url(r'^(?P<username>[a-zA-Z0-9_]+)/address/$', 'edit_address', name='edit-address'),
    url(r'^(?P<username>[a-zA-Z0-9_]+)/email/(?P<email>.*)$', 'delete_email', name='delete-email'),
    url(r'^(?P<username>[a-zA-Z0-9_]+)/projects/$', 'people_projects', name='user-projects'),
    url(r'^(?P<username>[a-zA-Z0-9_]+)/badges/$', 'people_badges', name='user-badges'),
)
