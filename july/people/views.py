from django.http import Http404, HttpResponseRedirect, HttpResponse
from django.core.urlresolvers import reverse
from django.shortcuts import render_to_response
from django.contrib.auth.decorators import login_required
from django.template.context import RequestContext
from django.template.defaultfilters import slugify

from gae_django.auth.models import User

from google.appengine.ext.ndb.query import Cursor
from google.appengine.ext import ndb, deferred

from july.people.models import Commit, Location, Project, Team
from july.cron import fix_location, fix_team

def people_projects(request, username):
    user = User.get_by_auth_id('own:%s' % username)
    if user == None:
        raise Http404("User not found")
    
    if getattr(user, 'projects', None) == None:
        projects = [] 
    else: 
        projects = user.projects
        projects = ndb.get_multi([Project.make_key(project) for project in projects])

    return render_to_response('people/people_projects.html', 
        {"projects":projects, 'profile':user}, 
        context_instance=RequestContext(request)) 

def user_profile(request, username):
    user = User.get_by_auth_id('own:%s' % username)
    if user == None:
        raise Http404("User not found")

    commits = Commit.query(ancestor=user.key).order(-Commit.timestamp).fetch(100)

    return render_to_response('people/profile.html', 
        {"commits":commits, 'profile':user}, 
        context_instance=RequestContext(request)) 

def leaderboard(request, template_name='people/leaderboard.html'):
    limit = 100
    cursor = request.GET.get('cursor')
    if cursor:
        cursor = Cursor(urlsafe=cursor)
    
    query = User.query().order(-ndb.GenericProperty('total'))
    models, next_cursor, more = query.fetch_page(limit, start_cursor=cursor)
    
    if next_cursor is not None:
        next_cursor = next_cursor.urlsafe()
    
    return render_to_response(template_name, 
                             {'next':next_cursor, 'more':more, 
                              'users':models},
                             context_instance=RequestContext(request)) 

def users_by_location(request, location_slug, 
                      template_name='people/people_list.html'):

    limit = 100
    cursor = request.GET.get('cursor')
    if cursor:
        cursor = Cursor(urlsafe=cursor)

    query = User.query(User.location_slug == location_slug)
    query = query.order(-ndb.GenericProperty('total'))
        
    models, next_cursor, more = query.fetch_page(limit, start_cursor=cursor)
    
    if next_cursor is not None:
        next_cursor = next_cursor.urlsafe()
    
    location = Location.get_by_id(location_slug)
    return render_to_response(template_name, 
                             {'next':next_cursor, 'more':more, 
                              'users':models,
                              'location': location, 'slug': location_slug}, 
                             context_instance=RequestContext(request)) 

def locations(request, template_name='people/locations.html'):
    
    limit = 100
    cursor = request.GET.get('cursor')
    if cursor:
        cursor = Cursor(urlsafe=cursor)
        
    query = Location.query().order(-Location.total)
    
    models, next_cursor, more = query.fetch_page(limit, start_cursor=cursor)
    
    if next_cursor is not None:
        next_cursor = next_cursor.urlsafe()
    
    return render_to_response(template_name,
                              {'locations': models, 'next': next_cursor,
                               'more': more},
                              context_instance=RequestContext(request))

def teams(request, template_name='people/teams.html'):

    limit = 100
    cursor = request.GET.get('cursor')
    if cursor:
        cursor = Cursor(urlsafe=cursor)
        
    query = Team.query().order(-Team.total)
    
    models, next_cursor, more = query.fetch_page(limit, start_cursor=cursor)
    
    if next_cursor is not None:
        next_cursor = next_cursor.urlsafe()
    
    return render_to_response(template_name,
                              {'next':next_cursor, 'more':more, 
                              'teams': models},
                              context_instance=RequestContext(request))

def team_details(request, team_slug, template_name='people/team_details.html'):
    limit = 100
    cursor = request.GET.get('cursor')
    if cursor:
        cursor = Cursor(urlsafe=cursor)

    query = User.query(ndb.GenericProperty('team_slug') == team_slug)
    query = query.order(-ndb.GenericProperty('total'))

    models, next_cursor, more = query.fetch_page(limit, start_cursor=cursor)
    
    if next_cursor is not None:
        next_cursor = next_cursor.urlsafe()
    
    team = Team.get_by_id(team_slug)
    return render_to_response(template_name, 
                             {'next':next_cursor, 'more':more, 
                              'users':models,
                              'team': team, 'slug': team_slug}, 
                             context_instance=RequestContext(request))

def projects(request, template_name='projects/index.html'):
    limit = 100
    cursor = request.GET.get('cursor')
    if cursor:
        cursor = Cursor(urlsafe=cursor)
    
    query = Project.query().order(-Project.total)
    
    models, next_cursor, more = query.fetch_page(limit, start_cursor=cursor)
    
    if next_cursor is not None:
        next_cursor = next_cursor.urlsafe()
    
    return render_to_response(template_name,
        {'projects': models, 'next': next_cursor, 'more': more},
        context_instance=RequestContext(request))

def project_details(request, slug, template_name='projects/details.html'):
    project_key = ndb.Key('Project', slug)
    project = project_key.get()
    if project is None:
        raise Http404("Project Not Found.")
    
    limit = 100
    cursor = request.GET.get('cursor')
    if cursor:
        cursor = Cursor(urlsafe=cursor)
    
    # TODO: pagination
    user_future = User.query().filter(ndb.GenericProperty('projects') == project.url).fetch_async(100)
    query = Commit.query().filter(Commit.project_slug == slug).order(-Commit.timestamp)
    
    commit_future = query.fetch_page_async(limit, start_cursor=cursor)
    
    commits, next_cursor, more = commit_future.get_result()
    users = user_future.get_result()
    
    if next_cursor is not None:
        next_cursor = next_cursor.urlsafe()
    
    return render_to_response(template_name,
        {'project': project, 'users': users, 'commits': commits,
         'next': next_cursor, 'more': more},
        context_instance=RequestContext(request))

@login_required
def edit_profile(request, username, template_name='people/edit.html'):
    from forms import EditUserForm
    user = User.get_by_auth_id('own:%s' % username)

    if user == None:
        raise Http404("User not found")
    
    if user.key != request.user.key:
        http403 = HttpResponse("This ain't you!")
        http403.status = 403
        return http403
    
    existing_slug = str(user.location_slug)
    existing_team = str(getattr(user, 'team_slug', ''))
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
        user.put()
        
        if user.location_slug != existing_slug:
            # Defer a couple tasks to update the locations
            deferred.defer(fix_location, str(user.location_slug))
            deferred.defer(fix_location, existing_slug)
            
        if getattr(user, 'team_slug', '') != existing_team:
            # Defer a couple tasks to update the teams
            deferred.defer(fix_team, str(user.team_slug))
            deferred.defer(fix_team, existing_team)
            
        return HttpResponseRedirect(
            reverse('member-profile', 
                    kwargs={'username':request.user.username}
                   )
        )
        
    

    return render_to_response(template_name, 
        {'form':form}, 
        context_instance=RequestContext(request))

@login_required
def delete_email(request, username, email):
    
    # the ID we are to delete
    auth_id = 'email:%s' % email
    
    user = User.get_by_auth_id('own:%s' % username)
    e_user = User.get_by_auth_id(auth_id)

    if user is None or e_user is None:
        raise Http404("User not found")
    
    if user != request.user or user != e_user:
        http403 = HttpResponse("This ain't you!")
        http403.status = 403
        return http403
    
    if request.method == "POST":
        # delete the email from the user
        user.auth_ids.remove(auth_id)
        user.unique_model.delete_multi(['User.auth_id:%s' % auth_id])
        user.put()
        return HttpResponseRedirect(
            reverse('member-profile', kwargs={'username':request.user.username})
        )
        
    

    return render_to_response('people/delete_email.html', 
        {'email': email}, 
        context_instance=RequestContext(request))
