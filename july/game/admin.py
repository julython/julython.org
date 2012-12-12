
from django.contrib import admin

from july.game.models import Game, Player, Board

class GameAdmin(admin.ModelAdmin):
    list_display = ['__unicode__', 'start', 'end']

class PlayerAdmin(admin.ModelAdmin):
    list_display = ['user', 'game', 'points']
    list_filter = ['game']

class BoardAdmin(admin.ModelAdmin):
    list_display = ['project', 'game', 'points']
    list_filter = ['game']

admin.site.register(Game, GameAdmin)
admin.site.register(Player, PlayerAdmin)
admin.site.register(Board, BoardAdmin)
