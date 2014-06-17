
from django.contrib import admin

from july.blog.models import Blog, Category


class BlogAdmin(admin.ModelAdmin):
    list_display = ['title', 'user', 'slug', 'posted', 'category']
    raw_id_fields = ['user']
    prepopulated_fields = {'slug': ['title']}

    def get_changeform_initial_data(self, request):
        # For Django 1.7
        initial = {}
        initial['user'] = request.user.pk
        return initial


class CategoryAdmin(admin.ModelAdmin):
    list_display = ['title', 'slug']


admin.site.register(Blog, BlogAdmin)
admin.site.register(Category, CategoryAdmin)
