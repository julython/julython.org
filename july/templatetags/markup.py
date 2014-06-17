from markdown import markdown

from django import template
from django.template.defaultfilters import stringfilter
from django.utils.encoding import force_unicode
from django.utils.safestring import mark_safe

from july.blog.models import Blog

register = template.Library()


@register.filter(is_safe=True)
@stringfilter
def markup(value):
    extensions = ["nl2br"]
    val = force_unicode(value)
    html = markdown(val, extensions, safe_mode=True, enable_attributes=False)
    return mark_safe(html)


@register.inclusion_tag('blog/blog_roll.html', takes_context=True)
def blog_roll(context):
    return {
        'blog': context['blog'],
        'posts': Blog.objects.all(),
    }
