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
            self.fields['about_me'].initial=user.about_me
            self.fields['url'].initial=user.url
            self.fields['facebook_url'].initial=user.facebook_url
            self.fields['email'].initial=user.email
        
        
