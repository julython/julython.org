import json
from settings import SECRET_KEY as SECRET

from django.utils.http import base36_to_int, int_to_base36
from django.utils.crypto import salted_hmac
from django.utils.translation import ugettext_lazy as _
from django.contrib.auth.decorators import login_required
from django.contrib.auth import login as auth_login
from django.shortcuts import render_to_response, render, redirect
from django.template import Context, loader
from django.contrib.sites.models import get_current_site
from django.core.mail import send_mail
from django.conf import settings
from django.http import HttpResponseRedirect

from july.game.models import Game
from july.forms import RegistrationForm
from july.models import User


def index(request):
    """Render the home page"""
    game = Game.active_or_latest()
    stats = game.histogram

    ctx = Context({
        'stats': json.dumps(stats),
        'game': game,
        'total': sum(stats),
        'user': request.user,
        'MEDIA_URL': settings.MEDIA_URL,
        'STATIC_URL': settings.STATIC_URL})

    return render_to_response('index.html', context_instance=ctx)


def help_view(request):
    """Render the help page"""
    ctx = Context({
        'user': request.user,
        'MEDIA_URL': settings.MEDIA_URL,
        'STATIC_URL': settings.STATIC_URL})

    return render_to_response('help.html', context_instance=ctx)


def register(request):
    """Register a new user"""
    def send_verification_email(user):
        token = salted_hmac(SECRET, user.email).hexdigest()
        site = get_current_site(request)
        domain = site.domain
        c = {
            'email': user.email,
            'domain': domain,
            'uid': int_to_base36(user.pk),
            'token': token
        }
        subject = _('Julython - verify your email')
        email = loader.render_to_string('registration/verify_email.html', c)
        send_mail(subject, email, None, [user.email])

    if request.POST:
        form = RegistrationForm(request.POST)
        if form.is_valid():
            user = form.save(commit=True)
            send_verification_email(user)
            # To login immediately after registering
            user.backend = "django.contrib.auth.backends.ModelBackend"
            auth_login(request, user)
            return redirect('july.views.index')
    else:
        form = RegistrationForm()

    return render(
        request,
        'registration/register.html', {'form': form})


@login_required
def login_redirect(request):
    if request.user.is_authenticated():
        return HttpResponseRedirect('/%s' % request.user.username)
    return HttpResponseRedirect('/')


def maintenance(request):
    """Site is down for maintenance, display this view for all."""
    ctx = Context({})

    return render_to_response('maintenance.html', context_instance=ctx)


def live(request):
    """Render the live view."""
    game = Game.active_or_latest()

    ctx = Context({
        'game': game,
        'user': request.user,
        'MEDIA_URL': settings.MEDIA_URL,
        'STATIC_URL': settings.STATIC_URL})

    return render_to_response('live/index.html', context_instance=ctx)


def email_verify(request, uidb36=None, token=None):
    """Verification for the user's email address"""
    assert uidb36 is not None and token is not None
    valid = False
    try:
        uid_int = base36_to_int(uidb36)
        user = User._default_manager.get(pk=uid_int)
    except (ValueError, OverflowError, User.DoesNotExist):
        user = None
    if user:
        expected_token = salted_hmac(SECRET, user.email).hexdigest()
        valid = expected_token == token
    if valid:
        user.verified = True
        user.save()
    return render(
        request, 'registration/email_verified.html', {'valid': valid})
