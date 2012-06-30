from django.http import Http404, HttpResponseRedirect, HttpResponse
from django.core.urlresolvers import reverse
from django.shortcuts import render_to_response
from django.contrib.auth.decorators import login_required
from django.template.context import RequestContext
from google.appengine.ext import ndb
from django.template.defaultfilters import slugify

from gae_django.auth.models import User

from july.people.models import Commit, Location

def user_profile(request, username):
    user = User.get_by_auth_id('twitter:%s' % username)
    if user == None:
        raise Http404("User not found")

    commits = Commit.query(ancestor=user.key).order(-Commit.timestamp).fetch(100)

    return render_to_response('people/profile.html', 
        {"commits":commits, 'profile':user}, 
        context_instance=RequestContext(request)) 

def users_by_location(request, location_slug, 
                      template_name='people/people_list.html'):

    users = User.query(User.location_slug == location_slug)
    users.order(-ndb.GenericProperty('total')).fetch(1000)

    location = Location.get_by_id(location_slug)
    
    return render_to_response(template_name, 
                             {'users':users, 'location': location, 'slug': location_slug}, 
                             context_instance=RequestContext(request)) 

def locations(request, template_name='people/locations.html'):

    locations = Location.query().order(-Location.total).fetch(1000)
    
    return render_to_response(template_name,
                              {'locations': locations},
                              context_instance=RequestContext(request))
 
@login_required
def edit_profile(request, username, template_name='people/edit.html'):
    from forms import EditUserForm
    user = User.get_by_auth_id('twitter:%s' % username)

    if user == None:
        raise Http404("User not found")
    
    if user.key != request.user.key:
        http403 = HttpResponse("This ain't you!")
        http403.status = 403
        return http403
    

    form = EditUserForm(request.POST or None, user=request.user)
    if form.is_valid():
        for key in form.cleaned_data:
            if key == 'email':
                continue
            setattr(user, key, form.cleaned_data.get(key))
        slugify(user.location)
        user.put()
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
    
    user = User.get_by_auth_id('twitter:%s' % username)
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
