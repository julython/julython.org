from django.contrib import admin

from models import Team, Commit, Location, Project

admin.site.register(Commit, 
    list_display=['hash', 'email', 'timestamp', 'project'], 
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
admin.site.register(Project, 
    list_display=["__unicode__", "url", "name", 'total'],
)