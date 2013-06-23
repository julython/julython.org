from django.conf.urls import patterns, url

from july.game import views


urlpatterns = patterns(
    'july.game.views',
    url(r'^people/$',
        views.PlayerList.as_view(),
        name='leaderboard'),
    url(r'^people/(?P<year>\d{4})/(?P<month>\d{1,2})/((?P<day>\d{1,2})/)?$',
        views.PlayerList.as_view(),
        name='leaderboard'),
    url(r'^teams/$',
        views.TeamCollection.as_view(),
        name='teams'),
    url(r'^teams/(?P<year>\d{4})/(?P<month>\d{1,2})/((?P<day>\d{1,2})/)?$',
        views.TeamCollection.as_view(),
        name='teams'),
    url(r'^teams/(?P<slug>[a-zA-Z0-9\-]+)/$',
        views.TeamView.as_view(),
        name='team-details'),
    url(r'^location/$',
        views.LocationCollection.as_view(),
        name='locations'),
    url(r'^location/(?P<year>\d{4})/(?P<month>\d{1,2})/((?P<day>\d{1,2})/)?$',
        views.LocationCollection.as_view(),
        name='locations'),
    url(r'^location/(?P<slug>[a-zA-Z0-9\-]+)/$',
        views.LocationView.as_view(),
        name='location-detail'),
    url(r'^projects/$',
        views.BoardList.as_view(),
        name='projects'),
    url(r'^projects/(?P<year>\d{4})/(?P<month>\d{1,2})/((?P<day>\d{1,2})/)?$',
        views.BoardList.as_view(),
        name='projects'),
    url(r'^projects/(?P<slug>.+)/$',
        views.ProjectView.as_view(),
        name='project-details'),
    # for local only debug purposes
    url(r'^events/(?P<action>pub|sub|ws)/(?P<channel>.*)$',
        'events', name='events'),
)
