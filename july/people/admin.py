from gae_django import admin

from gae_django.auth.models import User

from models import Team, Commit, Location

admin.site.register(User, 
    list_display=["last_name", "first_name", "username", "twitter"],
    list_filter=["is_superuser"], 
    exclude=["password", 'location_slug'],
    ordering=["last_name"],
    readonly_fields=['auth_ids'], 
    search_fields=["auth_ids", "first_name", "last_name"]
)
admin.site.register(Commit, 
    list_display=['hash', 'email', 'timestamp', 'project'], 
    exclude=['project_slug'],
    search_fields=['hash', 'email'], 
    ordering=['-timestamp']
)
admin.site.register(Location, 
    list_display=["__unicode__", "slug", 'total'],
    ordering=['-total']
)
admin.site.register(Team, 
    list_display=["__unicode__", "slug", 'total'],
    ordering=['-total']
)