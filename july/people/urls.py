from django.conf.urls.defaults import patterns, url

urlpatterns = patterns('july.people.views',
    url(r'^$', 'member_profile', name='member-profile'),
)
