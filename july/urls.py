from django.conf.urls.defaults import patterns, include, url
from django.contrib.auth import views
from gae_django import admin
from gae_django.auth.forms import AuthenticationForm

admin.autodiscover()

urlpatterns = patterns('',
    # Examples:
    url(r'^admin/', include(admin.site.urls)),
    url(r'^$', 'july.views.index', name='index'),
    url(r'^signin/$', views.login, {'authentication_form': AuthenticationForm}),
    url(r'^signout/$', views.logout),
    url(r'^accounts/', include('gae_django.auth.urls')),
    url(r'^me/', include('july.people.urls')),
)
