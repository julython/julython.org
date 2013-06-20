from django.http import Http404, HttpResponseRedirect, HttpResponse
from django.core.urlresolvers import reverse
from django.shortcuts import render_to_response, render, get_object_or_404
from django.contrib.auth.decorators import login_required
from django.template.context import RequestContext
from django.template.defaultfilters import slugify
from django.views.generic import detail
from django.utils.crypto import salted_hmac
from django.utils.translation import ugettext_lazy as _
from django.utils.http import int_to_base36
from django.utils.html import strip_tags
from django.core.mail import EmailMultiAlternatives
from django.template import loader
from django.contrib.sites.models import get_current_site
from social_auth.models import UserSocialAuth
from july.settings import SECRET_KEY as SECRET
from july.models import User


class UserProfile(detail.DetailView):
    model = User
    slug_field = 'username'
    context_object_name = 'profile'
    slug_url_kwarg = 'username'


# TODO (rmyers): move the rest of these views to knockback/backbone routes


def people_projects(request, username):
    user = get_object_or_404(User, username=username)

    return render_to_response('people/people_projects.html', {
            'projects': user.projects.all(),
            'profile': user,
            'active': 'projects',
        },
        context_instance=RequestContext(request))


def people_badges(request, username):
    user = get_object_or_404(User, username=username)

    return render_to_response('people/people_badges.html', {
            'badges': user.badges,
            'profile': user,
            'active': 'badges',
        },
        context_instance=RequestContext(request))


def send_verify_email(email, user_id, domain):
    token = salted_hmac(SECRET, email).hexdigest()
    c = {
        'email': email,
        'domain': domain,
        'uid': int_to_base36(user_id),
        'token': token
    }
    subject = _('Julython - verify your email')
    html = loader.render_to_string(
        'registration/verify_email.html', c)
    text = strip_tags(html)
    msg = EmailMultiAlternatives(subject, text, None, [email])
    msg.attach_alternative(html, 'text/html')
    msg.send()


@login_required
def edit_profile(request, username, template_name='people/edit.html'):
    from forms import EditUserForm
    user = request.user

    if user.username != request.user.username:
        http403 = HttpResponse("This ain't you!")
        http403.status = 403
        return http403

    form = EditUserForm(request.POST or None, user=request.user)

    if form.is_valid():
        for key, value in form.cleaned_data.iteritems():
            if key in ['gittip']:
                continue
            if key in ['email']:
                # send verification email
                domain = get_current_site(request).domain
                send_verify_email(value, user.pk, domain)
                # Don't actually add email to user model.
                continue
            if key == 'team':
                # slugify the team to allow easy lookups
                setattr(user, 'team_slug', slugify(value))
            setattr(user, key, value)
        user.save()

        return HttpResponseRedirect(
            reverse('member-profile', kwargs={'username': user.username}))

    ctx = {
        'form': form,
        'profile': user,
        'active': 'edit',
    }
    return render(request, template_name,
                  ctx, context_instance=RequestContext(request))


@login_required
def edit_address(request, username, template_name='people/edit_address.html'):
    from forms import EditAddressForm

    user = request.user

    if user.key != request.user.key:
        http403 = HttpResponse("This ain't you!")
        http403.status = 403
        return http403

    form = EditAddressForm(request.POST or None, user=user)

    if form.is_valid():
        for key, value in form.cleaned_data.iteritems():
            setattr(user, key, value)
            user.put()
        return HttpResponseRedirect(
            reverse('member-profile', kwargs={'username': user.username})
        )

    ctx = {
        'form': form,
        'profile': user,
        'active': 'edit',
    }
    return render(request, template_name, ctx,
                  context_instance=RequestContext(request))


@login_required
def delete_email(request, username, email):

    # the ID we are to delete
    user = User.objects.get(username=username)
    auth = UserSocialAuth.objects.get(provider="email", uid=email)
    e_user = auth.user

    if user is None or e_user is None:
        raise Http404("User not found")

    if user != request.user or user != e_user:
        http403 = HttpResponse("This ain't you!")
        http403.status = 403
        return http403

    if request.method == "POST":
        # delete the email from the user
        auth.delete()
        return HttpResponseRedirect(
            reverse('member-profile', kwargs={'username':request.user.username})
        )

    return render_to_response('people/delete_email.html',
        {'email': email},
        context_instance=RequestContext(request))
