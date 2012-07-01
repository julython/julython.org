from django import forms

class EditUserForm(forms.Form):
    # Match Twitter
    first_name = forms.CharField(max_length=255, required=True)
    last_name = forms.CharField(max_length=255, required=False)
    description = forms.CharField(label="About me", max_length=160, required=False,
        widget=forms.TextInput(attrs={'class': 'span6'}))
    url = forms.CharField(max_length=255, required=False,
        widget=forms.TextInput(attrs={'class': 'span4'}))
    location = forms.CharField(max_length=160, required=False)
    email = forms.EmailField(label="Add Email Address", required=False)

    def __init__(self, *args, **kwargs):
        self.user = kwargs.pop('user', None)
        self.emails = set([])
        super(EditUserForm, self).__init__(*args, **kwargs)
        if self.user:
            self.fields['first_name'].initial=getattr(self.user, 'first_name', None)
            self.fields['last_name'].initial=getattr(self.user, 'last_name', None)
            self.fields['description'].initial=getattr(self.user, 'description', None)
            self.fields['url'].initial=getattr(self.user, 'url', None)
            self.fields['location'].initial=getattr(self.user, 'location', None)
            # initialize the emails
            for auth in self.user.auth_ids:
                if auth.startswith('email'):
                    _, email = auth.split(':')
                    self.emails.add(email)
    
    def clean_email(self):
        email = self.cleaned_data['email']
        if not email:
            return None
        if email in self.emails:
            raise forms.ValidationError("You already have that email address!")
        
        # add the email address to the user, this will cause a ndb.put()
        added, _ = self.user.add_auth_id('email:%s' % email)
        if not added:
            raise forms.ValidationError("This email is already taken, if this is not right please email help@julython.org")
        
        return email
        
class CommitForm(forms.Form):
    
    message = forms.CharField(required=True)
    timestamp = forms.CharField(required=False)
    url = forms.URLField(verify_exists=False, required=False)
    email = forms.EmailField(required=False)
    author = forms.CharField(required=False)
    name = forms.CharField(required=False)
    hash = forms.CharField(required=False)
    
    def clean_timestamp(self):
        data = self.cleaned_data.get('timestamp')
        if data:
            import datetime
            data = datetime.datetime.fromtimestamp(float(data))
        return data

class ProjectForm(forms.Form):
    
    url = forms.URLField(required=True)
    forked = forms.BooleanField(required=False, initial=False)
    parent_url = forms.URLField(required=False)
    
    def clean_parent_url(self):
        data = self.cleaned_data
        if data['parent_url'] == '':
            return None
        return data['parent_url']
