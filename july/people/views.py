from django.http import Http404, HttpResponseRedirect, HttpResponse
from django.core.urlresolvers import reverse
from django.shortcuts import render_to_response, get_object_or_404
from django.contrib.auth.decorators import login_required
from django.template.context import RequestContext
from django.template.defaultfilters import slugify
from social_auth.models import UserSocialAuth

from july.models import User

from july.people.models import Commit, Location, Project, Team
#from july.cron import fix_location, fix_team

def people_projects(request, username):
    user = get_object_or_404(User, username=username)

    return render_to_response('people/people_projects.html', {
            'projects': user.projects.all(),
            'profile': user,
            'active': 'projects',
        },  
        context_instance=RequestContext(request)) 

def user_profile(request, username):
    user = get_object_or_404(User, username=username)

    commits = Commit.objects.filter(user=user).order_by('-timestamp')

    return render_to_response('people/profile.html', {
            'commits': commits, 
            'profile': user,
            'active': 'commits',
        }, 
        context_instance=RequestContext(request)) 

def leaderboard(request, template_name='people/leaderboard.html'):

    #TODO - limit, order and offset the users for this view.
    users = User.objects.order_by('-total')
    return render_to_response(template_name, 
                             { 'users':users},
                             context_instance=RequestContext(request)) 

def users_by_location(request, location_slug, 
                      template_name='people/people_list.html'):


    location = get_object_or_404(Location,slug=location_slug)
    users = location.location_members.order_by('-total')

    return render_to_response(template_name,
                             { 'users':users,
                              'location': location, 'slug': location_slug}, 
                             context_instance=RequestContext(request)) 

def locations(request, template_name='people/locations.html'):
    
    #TODO Sort by score for location (on LocationGame?)
    locations = Location.objects.order_by('-total')
    

    return render_to_response(template_name,
                              {'locations': locations},
                              context_instance=RequestContext(request))

def teams(request, template_name='people/teams.html'):
    #TODO - Make the teams ordered by score.
    teams = Team.objects.order_by('-total')

    return render_to_response(template_name,
                              {'teams': teams},
                              context_instance=RequestContext(request))

def team_details(request, team_slug, template_name='people/team_details.html'):

    team = get_object_or_404(Team, slug=team_slug)
    users = team.team_members.order_by('-total')
    
    return render_to_response(template_name,
                             { 'users':users,
                              'team': team, 'slug': team_slug}, 
                             context_instance=RequestContext(request))

def projects(request, template_name='projects/index.html'):

    projects = Project.objects.order_by('-total')


    return render_to_response(template_name,
        {'projects': projects},
        context_instance=RequestContext(request))

def project_details(request, slug, template_name='projects/details.html'):

    project = Project.objects.get(slug=slug)
    if project is None:
        raise Http404("Project Not Found.")
    

    # TODO: pagination
    users = project.user_set.all().order_by('-total')
    commits = project.commit_set.all().order_by('-timestamp')

    return render_to_response(template_name,
        {'project': project, 'users': users, 'commits': commits},
        context_instance=RequestContext(request))

@login_required
def edit_profile(request, username, template_name='people/edit.html'):
    from forms import EditUserForm
    from django.shortcuts import get_object_or_404
    user = request.user

    if user == None:
        raise Http404("User not found")
    
    if user.username != request.user.username:
        http403 = HttpResponse("This ain't you!")
        http403.status = 403
        return http403
    
    form = EditUserForm(request.POST or None, user=request.user)

    if form.is_valid():
        for key, value in form.cleaned_data.iteritems():
            if key == 'email':
                # Don't save the email to the profile
                continue
            if key == 'team':
                # slugify the team to allow easy lookups
                setattr(user, 'team_slug', slugify(value))
            setattr(user, key, value)
        user.save()

        return HttpResponseRedirect(
            reverse('member-profile',
                    kwargs={'username':request.user.username}
                   )
        )

    return render_to_response(template_name, {
            'form': form, 
            'profile': user,
            'active': 'edit',
        }, 
        context_instance=RequestContext(request))



@login_required
def edit_address(request, username, template_name='people/edit_address.html'):
    from forms import EditAddressForm

    user = request.user

    if user == None:
        raise Http404("User not found")

    if user.key != request.user.key:
        http403 = HttpResponse("This ain't you!")
        http403.status = 403
        return http403

    form = EditAddressForm(request.POST or None, user=user)

    if form.is_valid():
        for key, value in form.cleaned_data.iteritems():
            setattr(user,key,value)
            user.put()
        return HttpResponseRedirect(
            reverse('member-profile',
                kwargs={'username':request.user.username}
            )
        )
    

    return render_to_response(template_name, {
            'form': form, 
            'profile': user,
            'active': 'edit',
        },
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
