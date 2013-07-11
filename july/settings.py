import os
from django.conf.global_settings import TEMPLATE_CONTEXT_PROCESSORS as TCP

# Default settings that can be overwritten in secrets
DEBUG = True
SECRET_KEY = 'foobar'
DATABASE_ENGINE = 'django.db.backends.sqlite3'
DATABASE_NAME = 'julython.db'
DATABASE_PASSWORD = ''
DATABASE_SERVER = ''
DATABASE_USER = ''
LOGFILE_PATH = os.path.expanduser('~/julython.log')
TWITTER_CONSUMER_KEY = ''
TWITTER_CONSUMER_SECRET = ''
GITHUB_CONSUMER_KEY = ''
GITHUB_CONSUMER_SECRET = ''
GITHUB_APP_ID = GITHUB_CONSUMER_KEY
GITHUB_API_SECRET = GITHUB_CONSUMER_SECRET
EMAIL_HOST = '127.0.0.1'
EMAIL_PORT = '1025'

try:
    DEBUG = False
    from secrets import *
except ImportError:
    DEBUG = True

if DEBUG:
    import warnings
    warnings.filterwarnings(
        'error', r"DateTimeField received a naive datetime",
        RuntimeWarning, r'django\.db\.models\.fields')

CURRENT_DIR = os.path.abspath(os.path.dirname(__file__))

TEMPLATE_DEBUG = DEBUG

DEFAULT_FROM_EMAIL = 'Julython <mail@julython.org>'
SERVER_EMAIL = 'Julython <mail@julython.org>'
ADMINS = (
    ('Robert Myers', 'robert@julython.org'),
)

INTERNAL_IPS = ['127.0.0.1', 'localhost']

MANAGERS = ADMINS

DATABASES = {
    'default': {
        'ENGINE': DATABASE_ENGINE,
        'NAME': DATABASE_NAME,
        'USER': DATABASE_USER,
        'PASSWORD': DATABASE_PASSWORD,
        'HOST': DATABASE_SERVER,
        'PORT': '',
    }
}

# Local time zone for this installation. Choices can be found here:
# http://en.wikipedia.org/wiki/List_of_tz_zones_by_name
# although not all choices may be available on all operating systems.
# On Unix systems, a value of None will cause Django to use the same
# timezone as the operating system.
# If running in a Windows environment this must be set to the same as your
# system time zone.
TIME_ZONE = 'UTC'

# Language code for this installation. All choices can be found here:
# http://www.i18nguy.com/unicode/language-identifiers.html
LANGUAGE_CODE = 'en-us'

SITE_ID = 1

# Timezone Support
USE_TZ = True

# If you set this to False, Django will make some optimizations so as not
# to load the internationalization machinery.
USE_I18N = True

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
    #'django.contrib.staticfiles.finders.DefaultStorageFinder',
)

# List of callables that know how to import templates from various sources.
TEMPLATE_LOADERS = (
    'django.template.loaders.filesystem.Loader',
    'django.template.loaders.app_directories.Loader',
)

MIDDLEWARE_CLASSES = (
    'django.middleware.common.CommonMiddleware',
    'django.middleware.csrf.CsrfViewMiddleware',
    'django.contrib.sessions.middleware.SessionMiddleware',
    'django.middleware.locale.LocaleMiddleware',
    'django.contrib.auth.middleware.AuthenticationMiddleware',
    'django.contrib.messages.middleware.MessageMiddleware',
    'july.middleware.AbuseMiddleware',
    'debug_toolbar.middleware.DebugToolbarMiddleware',
)

TEMPLATE_CONTEXT_PROCESSORS = TCP + (
    'django.core.context_processors.request',
)

DEBUG_TOOLBAR_CONFIG = {
    'INTERCEPT_REDIRECTS': True,
    'HIDE_DJANGO_SQL': True,
    'ENABLE_STACKTRACES': True,
}

ROOT_URLCONF = 'july.urls'

TEMPLATE_DIRS = (
    # Put strings here, like "/home/html/django_templates"
    # Always use forward slashes, even on Windows.
    # Don't forget to use absolute paths, not relative paths.
)

INSTALLED_APPS = (
    'july',
    'july.game',
    'july.people',
    'django.contrib.auth',
    'django.contrib.sessions',
    'django.contrib.staticfiles',
    'django.contrib.messages',
    'django.contrib.admin',
    'django.contrib.admindocs',
    'django.contrib.contenttypes',
    'debug_toolbar',
    'social_auth',
    'south',
)

AUTHENTICATION_BACKENDS = [
    'july.auth.twitter.TwitterBackend',
    'july.auth.github.GithubBackend',
    'django.contrib.auth.backends.ModelBackend',
]

SESSION_ENGINE = 'django.contrib.sessions.backends.signed_cookies'
MESSAGE_STORAGE = 'django.contrib.messages.storage.session.SessionStorage'
SESSION_SAVE_EVERY_REQUEST = True

HCACHES = {
    'default': {
        'BACKEND': 'django.core.cache.backends.memcached.MemcachedCache',
        'TIMEOUT': 600,
    }
}

# Django 1.5 Custom User Model !! ftw
AUTH_USER_MODEL = 'july.User'
SOCIAL_AUTH_USER_MODEL = AUTH_USER_MODEL

SOCIAL_AUTH_DEFAULT_USERNAME = 'new_social_auth_user'
SOCIAL_AUTH_UUID_LENGTH = 3
SOCIAL_AUTH_PROTECTED_USER_FIELDS = ['email', 'location', 'url', 'description']
SOCIAL_AUTH_COMPLETE_URL_NAME = 'socialauth_complete'
SOCIAL_AUTH_ASSOCIATE_URL_NAME = 'socialauth_associate_complete'

# Just so we can use the same names for variables - why different social_auth??
GITHUB_APP_ID = GITHUB_CONSUMER_KEY
GITHUB_API_SECRET = GITHUB_CONSUMER_SECRET
GITHUB_EXTENDED_PERMISSIONS = ['user', 'public_repo']
TWITTER_EXTRA_DATA = [('screen_name', 'screen_name')]

ABUSE_LIMIT = 3

LOGGING = {
    'version': 1,
    'disable_existing_loggers': False,
    'formatters': {
        'verbose': {
            'format': '%(levelname)s %(asctime)s %(module)s %(message)s'
        },
        'simple': {
            'format': '%(levelname)s %(message)s'
        },
    },
    'handlers': {
        'null': {
            'level': 'DEBUG',
            'class': 'django.utils.log.NullHandler',
        },
        'console': {
            'level': 'DEBUG',
            'class': 'logging.StreamHandler',
            'formatter': 'simple'
        },
        'file': {
            'level': 'INFO',
            'class': 'logging.handlers.RotatingFileHandler',
            'formatter': 'simple',
            'maxBytes': 100000000,
            'backupCount': 3,
            'filename': LOGFILE_PATH,
        },
    },
    'loggers': {
        '': {
            'handlers': ['console', 'file'],
            'propagate': True,
            'level': 'INFO',
        },
    }
}
