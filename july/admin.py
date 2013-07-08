from django.contrib import admin

from july.models import User
from social_auth.models import UserSocialAuth


class AuthInline(admin.TabularInline):
    model = UserSocialAuth


class UserAdmin(admin.ModelAdmin):
    list_display = ['username', 'email', 'location', 'team']
    search_fields = ['username', 'email']
    inlines = [AuthInline]
    raw_id_fields = ['projects', 'location', 'team']
    list_filter = ['is_active', 'is_superuser']


admin.site.register(User, UserAdmin)
