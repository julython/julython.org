from django.contrib import admin

from models import Team, Commit, Location, AchievedBadge, Badge


admin.site.register(
    Commit,
    list_display=['hash', 'email', 'timestamp', 'project', 'user'],
    search_fields=['hash', 'email', 'project__name', 'user__username'],
    ordering=['-timestamp']
)


admin.site.register(
    Location,
    list_display=["__unicode__", "slug", 'total'],
    ordering=['-total']
)


admin.site.register(
    Team,
    list_display=["__unicode__", "slug", 'total'],
    ordering=['-total']
)


admin.site.register(AchievedBadge)


admin.site.register(
    Badge,
    list_display=["__unicode__"]
)
