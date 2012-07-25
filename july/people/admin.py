from gae_django import admin

from gae_django.auth.models import User

from models import Accumulator, Commit, Location

admin.site.register(User, list_display=["get_full_name", "username", "twitter"],
    list_filter=["is_superuser"], exclude=["password", 'location_slug'], readonly_fields=['auth_ids'])
admin.site.register(Commit, exclude=['project_slug'])
admin.site.register(Location, list_display=["slug", 'total'])