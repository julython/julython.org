# -*- coding: utf-8 -*-
import datetime
from south.db import db
from south.v2 import SchemaMigration
from django.db import models


class Migration(SchemaMigration):

    def forwards(self, orm):
        # Adding model 'LanguageBoard'
        db.create_table(u'game_languageboard', (
            (u'id', self.gf('django.db.models.fields.AutoField')(primary_key=True)),
            ('game', self.gf('django.db.models.fields.related.ForeignKey')(to=orm['game.Game'])),
            ('points', self.gf('django.db.models.fields.IntegerField')(default=0)),
            ('language', self.gf('django.db.models.fields.related.ForeignKey')(to=orm['people.Language'])),
        ))
        db.send_create_signal(u'game', ['LanguageBoard'])

        # Adding M2M table for field language_boards on 'Player'
        m2m_table_name = db.shorten_name(u'game_player_language_boards')
        db.create_table(m2m_table_name, (
            ('id', models.AutoField(verbose_name='ID', primary_key=True, auto_created=True)),
            ('player', models.ForeignKey(orm[u'game.player'], null=False)),
            ('languageboard', models.ForeignKey(orm[u'game.languageboard'], null=False))
        ))
        db.create_unique(m2m_table_name, ['player_id', 'languageboard_id'])


    def backwards(self, orm):
        # Deleting model 'LanguageBoard'
        db.delete_table(u'game_languageboard')

        # Removing M2M table for field language_boards on 'Player'
        db.delete_table(db.shorten_name(u'game_player_language_boards'))


    models = {
        u'auth.group': {
            'Meta': {'object_name': 'Group'},
            u'id': ('django.db.models.fields.AutoField', [], {'primary_key': 'True'}),
            'name': ('django.db.models.fields.CharField', [], {'unique': 'True', 'max_length': '80'}),
            'permissions': ('django.db.models.fields.related.ManyToManyField', [], {'to': u"orm['auth.Permission']", 'symmetrical': 'False', 'blank': 'True'})
        },
        u'auth.permission': {
            'Meta': {'ordering': "(u'content_type__app_label', u'content_type__model', u'codename')", 'unique_together': "((u'content_type', u'codename'),)", 'object_name': 'Permission'},
            'codename': ('django.db.models.fields.CharField', [], {'max_length': '100'}),
            'content_type': ('django.db.models.fields.related.ForeignKey', [], {'to': u"orm['contenttypes.ContentType']"}),
            u'id': ('django.db.models.fields.AutoField', [], {'primary_key': 'True'}),
            'name': ('django.db.models.fields.CharField', [], {'max_length': '50'})
        },
        u'contenttypes.contenttype': {
            'Meta': {'ordering': "('name',)", 'unique_together': "(('app_label', 'model'),)", 'object_name': 'ContentType', 'db_table': "'django_content_type'"},
            'app_label': ('django.db.models.fields.CharField', [], {'max_length': '100'}),
            u'id': ('django.db.models.fields.AutoField', [], {'primary_key': 'True'}),
            'model': ('django.db.models.fields.CharField', [], {'max_length': '100'}),
            'name': ('django.db.models.fields.CharField', [], {'max_length': '100'})
        },
        u'game.board': {
            'Meta': {'ordering': "['-points']", 'object_name': 'Board'},
            'game': ('django.db.models.fields.related.ForeignKey', [], {'to': u"orm['game.Game']"}),
            u'id': ('django.db.models.fields.AutoField', [], {'primary_key': 'True'}),
            'points': ('django.db.models.fields.IntegerField', [], {'default': '0'}),
            'project': ('django.db.models.fields.related.ForeignKey', [], {'to': u"orm['people.Project']"})
        },
        u'game.game': {
            'Meta': {'ordering': "['-end']", 'object_name': 'Game'},
            'boards': ('django.db.models.fields.related.ManyToManyField', [], {'to': u"orm['people.Project']", 'through': u"orm['game.Board']", 'symmetrical': 'False'}),
            'commit_points': ('django.db.models.fields.IntegerField', [], {'default': '1'}),
            'end': ('django.db.models.fields.DateTimeField', [], {}),
            u'id': ('django.db.models.fields.AutoField', [], {'primary_key': 'True'}),
            'language_boards': ('django.db.models.fields.related.ManyToManyField', [], {'to': u"orm['people.Language']", 'through': u"orm['game.LanguageBoard']", 'symmetrical': 'False'}),
            'players': ('django.db.models.fields.related.ManyToManyField', [], {'to': u"orm['july.User']", 'through': u"orm['game.Player']", 'symmetrical': 'False'}),
            'problem_points': ('django.db.models.fields.IntegerField', [], {'default': '5'}),
            'project_points': ('django.db.models.fields.IntegerField', [], {'default': '10'}),
            'start': ('django.db.models.fields.DateTimeField', [], {})
        },
        u'game.languageboard': {
            'Meta': {'ordering': "['-points']", 'object_name': 'LanguageBoard'},
            'game': ('django.db.models.fields.related.ForeignKey', [], {'to': u"orm['game.Game']"}),
            u'id': ('django.db.models.fields.AutoField', [], {'primary_key': 'True'}),
            'language': ('django.db.models.fields.related.ForeignKey', [], {'to': u"orm['people.Language']"}),
            'points': ('django.db.models.fields.IntegerField', [], {'default': '0'})
        },
        u'game.player': {
            'Meta': {'ordering': "['-points']", 'object_name': 'Player'},
            'boards': ('django.db.models.fields.related.ManyToManyField', [], {'to': u"orm['game.Board']", 'symmetrical': 'False'}),
            'game': ('django.db.models.fields.related.ForeignKey', [], {'to': u"orm['game.Game']"}),
            u'id': ('django.db.models.fields.AutoField', [], {'primary_key': 'True'}),
            'language_boards': ('django.db.models.fields.related.ManyToManyField', [], {'to': u"orm['game.LanguageBoard']", 'symmetrical': 'False'}),
            'points': ('django.db.models.fields.IntegerField', [], {'default': '0'}),
            'user': ('django.db.models.fields.related.ForeignKey', [], {'to': u"orm['july.User']"})
        },
        u'july.user': {
            'Meta': {'object_name': 'User'},
            'date_joined': ('django.db.models.fields.DateTimeField', [], {'default': 'datetime.datetime.now'}),
            'description': ('django.db.models.fields.TextField', [], {'blank': 'True'}),
            'email': ('django.db.models.fields.EmailField', [], {'max_length': '75', 'blank': 'True'}),
            'first_name': ('django.db.models.fields.CharField', [], {'max_length': '30', 'blank': 'True'}),
            'groups': ('django.db.models.fields.related.ManyToManyField', [], {'to': u"orm['auth.Group']", 'symmetrical': 'False', 'blank': 'True'}),
            u'id': ('django.db.models.fields.AutoField', [], {'primary_key': 'True'}),
            'is_active': ('django.db.models.fields.BooleanField', [], {'default': 'True'}),
            'is_staff': ('django.db.models.fields.BooleanField', [], {'default': 'False'}),
            'is_superuser': ('django.db.models.fields.BooleanField', [], {'default': 'False'}),
            'last_login': ('django.db.models.fields.DateTimeField', [], {'default': 'datetime.datetime.now'}),
            'last_name': ('django.db.models.fields.CharField', [], {'max_length': '30', 'blank': 'True'}),
            'location': ('django.db.models.fields.related.ForeignKey', [], {'blank': 'True', 'related_name': "'location_members'", 'null': 'True', 'to': u"orm['people.Location']"}),
            'password': ('django.db.models.fields.CharField', [], {'max_length': '128'}),
            'picture_url': ('django.db.models.fields.URLField', [], {'max_length': '200', 'null': 'True', 'blank': 'True'}),
            'projects': ('django.db.models.fields.related.ManyToManyField', [], {'symmetrical': 'False', 'to': u"orm['people.Project']", 'null': 'True', 'blank': 'True'}),
            'team': ('django.db.models.fields.related.ForeignKey', [], {'blank': 'True', 'related_name': "'team_members'", 'null': 'True', 'to': u"orm['people.Team']"}),
            'url': ('django.db.models.fields.URLField', [], {'max_length': '200', 'null': 'True', 'blank': 'True'}),
            'user_permissions': ('django.db.models.fields.related.ManyToManyField', [], {'to': u"orm['auth.Permission']", 'symmetrical': 'False', 'blank': 'True'}),
            'username': ('django.db.models.fields.CharField', [], {'unique': 'True', 'max_length': '30'})
        },
        u'people.language': {
            'Meta': {'object_name': 'Language'},
            'name': ('django.db.models.fields.CharField', [], {'max_length': '64', 'primary_key': 'True'})
        },
        u'people.location': {
            'Meta': {'object_name': 'Location'},
            'name': ('django.db.models.fields.CharField', [], {'max_length': '64'}),
            'slug': ('django.db.models.fields.SlugField', [], {'max_length': '50', 'primary_key': 'True'}),
            'total': ('django.db.models.fields.IntegerField', [], {'default': '0'})
        },
        u'people.project': {
            'Meta': {'object_name': 'Project'},
            'created_on': ('django.db.models.fields.DateTimeField', [], {'auto_now_add': 'True', 'blank': 'True'}),
            'description': ('django.db.models.fields.TextField', [], {'blank': 'True'}),
            'forked': ('django.db.models.fields.BooleanField', [], {'default': 'False'}),
            'forks': ('django.db.models.fields.IntegerField', [], {'default': '0'}),
            u'id': ('django.db.models.fields.AutoField', [], {'primary_key': 'True'}),
            'languages': ('django.db.models.fields.related.ManyToManyField', [], {'to': u"orm['people.Language']", 'symmetrical': 'False', 'blank': 'True'}),
            'name': ('django.db.models.fields.CharField', [], {'max_length': '255', 'blank': 'True'}),
            'parent_url': ('django.db.models.fields.CharField', [], {'max_length': '255', 'blank': 'True'}),
            'repo_id': ('django.db.models.fields.IntegerField', [], {'null': 'True', 'blank': 'True'}),
            'service': ('django.db.models.fields.CharField', [], {'default': "''", 'max_length': '30', 'blank': 'True'}),
            'slug': ('django.db.models.fields.SlugField', [], {'max_length': '50'}),
            'url': ('django.db.models.fields.CharField', [], {'max_length': '255'}),
            'watchers': ('django.db.models.fields.IntegerField', [], {'default': '0'})
        },
        u'people.team': {
            'Meta': {'object_name': 'Team'},
            'name': ('django.db.models.fields.CharField', [], {'max_length': '64'}),
            'slug': ('django.db.models.fields.SlugField', [], {'max_length': '50', 'primary_key': 'True'}),
            'total': ('django.db.models.fields.IntegerField', [], {'default': '0'})
        }
    }

    complete_apps = ['game']