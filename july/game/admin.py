
from django.contrib import admin

from july.game.models import Game, Player

class PlayerInline(admin.TabularInline):
    model = Player

class GameAdmin(admin.ModelAdmin):
    inlines = [PlayerInline]

admin.site.register(Game, GameAdmin)
