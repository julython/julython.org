from django.contrib import admin
from models import Team, Commit, Location, AchievedBadge
from models import Badge, Language, Extension


class ExtensionInline(admin.TabularInline):
    model = Extension

admin.site.register(
    Commit,
    list_display=['hash', 'email', 'timestamp', 'project', 'user'],
    search_fields=['hash', 'email', 'project__name', 'user__username'],
    ordering=['-timestamp'])

admin.site.register(
    Location,
    list_display=["__unicode__", "slug", 'total'],
    ordering=['-total'])

admin.site.register(
    Language,
    list_display=["__unicode__"],
    inlines=[ExtensionInline])

admin.site.register(
    Team,
    list_display=["__unicode__", "slug", 'total'],
    ordering=['-total'])

admin.site.register(
    Badge,
    list_display=["__unicode__"])

admin.site.register(AchievedBadge)
