from django.contrib import admin

from july.models import User
from social_auth.models import UserSocialAuth


class AuthInline(admin.TabularInline):
    model = UserSocialAuth


def purge_commits(modeladmin, request, queryset):
    for obj in queryset:
        obj.commit_set.all().delete()
purge_commits.short_description = "Purge Commits"


class UserAdmin(admin.ModelAdmin):
    list_display = ['username', 'email', 'location', 'team']
    search_fields = ['username', 'email']
    inlines = [AuthInline]
    raw_id_fields = ['projects', 'location', 'team']
    list_filter = ['is_active', 'is_superuser']
    actions = [purge_commits]


admin.site.register(User, UserAdmin)
