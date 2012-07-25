from django.conf.urls.defaults import patterns, include, url
from django.contrib.auth import views
from gae_django import admin
from gae_django.auth.forms import AuthenticationForm

admin.autodiscover()

urlpatterns = patterns('',
    # This line should only be active during maintenance!
    #url(r'^.*', 'july.views.maintenance'),
    url(r'^_admin/', admin.site.urls),
    url(r'^_ah/warmup', 'july.views.warmup'),
    url(r'^$', 'july.views.index', name='index'),
    url(r'^live/', 'july.live.views.live', name='julython-live'),
    url(r'^_reconnect/', 'july.live.views.reconnect', name='live-reconnect'),
    url(r'^help/', 'july.views.help_view', name='help'),
    url(r'^signin/$', views.login, {'authentication_form': AuthenticationForm}, name="signin"),
    url(r'^signout/$', views.logout, {'next_page': '/'}, name="signout"),
    url(r'^accounts/profile', 'july.views.login_redirect'),
    url(r'^accounts/', include('gae_django.auth.urls')),
    url(r'^', include('july.people.urls')),

)
