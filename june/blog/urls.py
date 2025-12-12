from django.conf.urls import patterns, url
from django.views.generic.detail import DetailView
from django.views.generic.list import ListView

from july.blog.models import Blog

urlpatterns = patterns(
    '',
    url(r'^blog/$', ListView.as_view(model=Blog),
        name="blog"),
    url(r'^blog/(?P<slug>[-_\w]+)/$', DetailView.as_view(model=Blog),
        name="view_blog_post"),
)
