from django import forms

class EditUserForm(forms.Form):
    about_me = forms.CharField(widget=forms.Textarea, required=False)
    url = forms.CharField(max_length=255, required=False)
    facebook_url = forms.CharField(max_length=255, required=False)
    email = forms.EmailField(max_length=255)

    def __init__(self, *args, **kwargs):
        user = kwargs.pop('user', None)
        super(EditUserForm, self).__init__(*args, **kwargs)
        if user:
            self.fields['about_me'].initial=getattr(user, 'about_me', None)
            self.fields['url'].initial=getattr(user, 'url', None)
            self.fields['facebook_url'].initial=getattr(user, 'facebook_url', None)
            self.fields['email'].initial=user.email
        
        
