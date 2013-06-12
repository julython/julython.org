from django.conf.urls import patterns, include, url
from django.contrib.auth import views
from django.contrib import admin

from tastypie.api import Api

from july import api

v1_api = Api(api_name='v1')
v1_api.register(api.CommitResource())
v1_api.register(api.ProjectResource())
v1_api.register(api.UserResource())
v1_api.register(api.LocationResource())
v1_api.register(api.TeamResource())

admin.autodiscover()


urlpatterns = patterns(
    '',
    # This line should only be active during maintenance!
    #url(r'^.*', 'july.views.maintenance'),
    url(r'^admin/doc/', include('django.contrib.admindocs.urls')),
    url(r'^_admin/', admin.site.urls),
    # bitbucket and github are special apis
    url(r'^api/v1/bitbucket', api.BitbucketHandler.as_view()),
    url(r'^api/v1/github', api.GithubHandler.as_view()),
    url(r'^api/', include(v1_api.urls)),
    url(r'^$', 'july.views.index', name='index'),
    url(r'^live/', 'july.views.live', name='julython-live'),
    # for local only debug purposes
    url(r'^events/(?P<action>pub|sub|ws)/(?P<channel>.*)$',
        'july.live.views.events', name='events'),
    url(r'^help/', 'july.views.help_view', name='help'),
    url(r'^signin/$', views.login, name="signin"),
    url(r'^register/$', 'july.views.register', name="register"),
    url(r'^signout/$', views.logout, {'next_page': '/'}, name="signout"),
    url(r'^accounts/profile', 'july.views.login_redirect'),
    url(r'^accounts/', include('social_auth.urls')),
    url(r'^', include('july.game.urls')),
    url(r'^', include('july.people.urls')),

)
