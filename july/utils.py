# coding: utf-8
import requests

def check_location(location):
    resp = requests.get('http://maps.googleapis.com/maps/api/geocode/json',
        params={'address': location, 'sensor': 'false'})
    resp.raise_for_status()

    data = resp.json()
    if data['status'] == 'ZERO_RESULTS':
        return None
    
    try:
        location = data['results'][0]['address_components']
    except (KeyError, IndexError):
        return None
    
    res = []
    for component in location:
        if 'locality' in component['types']:
            res.append(component['long_name'])
        if 'administrative_area_level_1' in component['types']:
            res.append(component['long_name'])
        if 'country' in component['types']:
            res.append(component['long_name'])

    return res


