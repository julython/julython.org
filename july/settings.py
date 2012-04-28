# Django settings for men project.
import os
from secrets import *

CURRENT_DIR = os.path.abspath(os.path.dirname(__file__))

DEBUG = True
TEMPLATE_DEBUG = DEBUG

ADMINS = (
    ('Robert Myers', 'robert@julython.org'),
)

INTERNAL_IPS = ['127.0.0.1', 'localhost']

MANAGERS = ADMINS

DATABASES = {
    'default': {
        'ENGINE': 'gae_django.db.gae', # Add 'postgresql_psycopg2', 'postgresql', 'mysql', 'sqlite3' or 'oracle'.
        'NAME': '',                      # Or path to database file if using sqlite3.
        'USER': '',                      # Not used with sqlite3.
        'PASSWORD': '',                  # Not used with sqlite3.
        'HOST': '',                      # Set to empty string for localhost. Not used with sqlite3.
        'PORT': '',                      # Set to empty string for default. Not used with sqlite3.
    }
}

# Local time zone for this installation. Choices can be found here:
# http://en.wikipedia.org/wiki/List_of_tz_zones_by_name
# although not all choices may be available on all operating systems.
# On Unix systems, a value of None will cause Django to use the same
# timezone as the operating system.
# If running in a Windows environment this must be set to the same as your
# system time zone.
TIME_ZONE = 'America/Chicago'

# Language code for this installation. All choices can be found here:
# http://www.i18nguy.com/unicode/language-identifiers.html
LANGUAGE_CODE = 'en-us'

SITE_ID = 1

# If you set this to False, Django will make some optimizations so as not
# to load the internationalization machinery.
USE_I18N = False

# If you set this to False, Django will not format dates, numbers and
# calendars according to the current locale
USE_L10N = False

# Absolute filesystem path to the directory that will hold user-uploaded files.
# Example: "/home/media/media.lawrence.com/media/"
MEDIA_ROOT = ''

# URL that handles the media served from MEDIA_ROOT. Make sure to use a
# trailing slash.
# Examples: "http://media.lawrence.com/media/", "http://example.com/media/"
MEDIA_URL = '/media/'

# Absolute path to the directory static files should be collected to.
# Don't put anything in this directory yourself; store your static files
# in apps' "static/" subdirectories and in STATICFILES_DIRS.
# Example: "/home/media/media.lawrence.com/static/"
STATIC_ROOT = os.path.join(CURRENT_DIR, 'static_root')

# URL prefix for static files.
# Example: "http://media.lawrence.com/static/"
STATIC_URL = '/static/'

# URL prefix for admin static files -- CSS, JavaScript and images.
# Make sure to use a trailing slash.
# Examples: "http://foo.com/static/admin/", "/static/admin/".
ADMIN_MEDIA_PREFIX = '/static/admin/'

# Additional locations of static files
STATICFILES_DIRS = (
    os.path.join(CURRENT_DIR, 'static'),
    # Put strings here, like "/home/html/static" or "C:/www/django/static".
    # Always use forward slashes, even on Windows.
    # Don't forget to use absolute paths, not relative paths.
)

# List of finder classes that know how to find static files in
# various locations.
STATICFILES_FINDERS = (
    'django.contrib.staticfiles.finders.FileSystemFinder',
    'django.contrib.staticfiles.finders.AppDirectoriesFinder',
#    'django.contrib.staticfiles.finders.DefaultStorageFinder',
)

# List of callables that know how to import templates from various sources.
TEMPLATE_LOADERS = (
    'django.template.loaders.filesystem.Loader',
    'django.template.loaders.app_directories.Loader',
#     'django.template.loaders.eggs.Loader',
)

MIDDLEWARE_CLASSES = (
    'django.middleware.common.CommonMiddleware',
    'django.contrib.sessions.middleware.SessionMiddleware',
    'django.middleware.csrf.CsrfViewMiddleware',
    'django.contrib.auth.middleware.AuthenticationMiddleware',
    'django.contrib.messages.middleware.MessageMiddleware',
    'debug_toolbar.middleware.DebugToolbarMiddleware',
)

DEBUG_TOOLBAR_PANELS = (
    'debug_toolbar.panels.version.VersionDebugPanel',
    'debug_toolbar.panels.timer.TimerDebugPanel',
    'debug_toolbar.panels.settings_vars.SettingsVarsDebugPanel',
    'debug_toolbar.panels.headers.HeaderDebugPanel',
    'debug_toolbar.panels.request_vars.RequestVarsDebugPanel',
    'debug_toolbar.panels.template.TemplateDebugPanel',
    'gae_django.toolbar.panel.AppStatsPanel',
    'debug_toolbar.panels.signals.SignalDebugPanel',
    'debug_toolbar.panels.logger.LoggingPanel',
)

DEBUG_TOOLBAR_CONFIG = {
    'INTERCEPT_REDIRECTS': True,
    'HIDE_DJANGO_SQL': True,
    'ENABLE_STACKTRACES' : True,
}

ROOT_URLCONF = 'july.urls'

TEMPLATE_DIRS = (
    # Put strings here, like "/home/html/django_templates" or "C:/www/django/templates".
    # Always use forward slashes, even on Windows.
    # Don't forget to use absolute paths, not relative paths.
)

INSTALLED_APPS = (
    'django.contrib.auth',
    'django.contrib.sessions',
    'django.contrib.staticfiles',
    'django.contrib.messages',
    'django.contrib.admin',
    'debug_toolbar',
    'gae_django.toolbar',
    'gae_django.admin',
    'gae_django.auth',
    'july',
)

AUTHENTICATION_BACKENDS = ['gae_django.auth.backend.GAEBackend', 'gae_django.auth.backend.GAETwitterBackend']
SESSION_ENGINE = 'django.contrib.sessions.backends.cache'
MESSAGE_STORAGE = 'django.contrib.messages.storage.session.SessionStorage'
SESSION_SAVE_EVERY_REQUEST = True

CACHES = {
    'default': {
        'BACKEND': 'django.core.cache.backends.memcached.MemcachedCache',
        'TIMEOUT': 3600*24*2, # Two Weeks
    }
}


# SETUP local and prod settings
VERSION = os.environ.get('CURRENT_VERSION_ID', '1.1')

# TODO: possibly break this up into separate files if needed.
if VERSION == '1.1':
    # We are running locally
    MAIN_URL = 'http://localhost:8080/'
    DEBUG = True
    TEMPLATE_DEBUG = DEBUG
    try:
        # allow developers to override url or other settings.
        from settings_local import *
    except:
        pass
else:
    MAIN_URL = 'http://www.julython.org/'

TWITTER_CALLBACK = '%saccounts/twitter/verify/' % MAIN_URL


LOGIN_REDIRECT_URL = '/'