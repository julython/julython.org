from django.conf.urls import patterns, url

from july.people import views


urlpatterns = patterns(
    'july.people.views',
    url(r'^(?P<username>[\w.@+-]+)/$',
        views.UserProfile.as_view(),
        name='member-profile'),
    url(r'^(?P<username>[\w.@+-]+)/edit/$',
        'edit_profile', name='edit-profile'),
    url(r'^(?P<username>[\w.@+-]+)/address/$',
        'edit_address', name='edit-address'),
    url(r'^(?P<username>[\w.@+-]+)/email/(?P<email>.*)$',
        'delete_email', name='delete-email'),
    url(r'^(?P<username>[\w.@+-]+)/project/(?P<slug>.*)$',
        'delete_project', name='delete-project'),
    url(r'^(?P<username>[\w.@+-]+)/projects/$',
        'people_projects', name='user-projects'),
)
