from django.contrib import admin

from july.models import User

admin.site.register(User, 
    list_display=['username', 'email', 'location', 'team'], 
    search_fields=['username', 'email'],
)