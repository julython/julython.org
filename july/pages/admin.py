
from gae_django import admin

from models import Section

admin.site.register(Section, list_display=['title', 'order'])