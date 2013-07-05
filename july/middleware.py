import datetime
from datetime import date
from django.conf import settings

oneday = datetime.timedelta(days=1)
ABUSE_DELTA = datetime.timedelta(days=settings.ABUSE_LIMIT)

class AbuseMiddleware(object):
    def _can_report_abuse(self, request):
        def can_report_abuse():
            abuse_date = request.session.get('abuse_date')
            return not abuse_date or abuse_date < date.today()
        return can_report_abuse

    def _abuse_reported(self, request):
        def abuse_reported():
            abuse_date = request.session.get('abuse_date')
            if not abuse_date or abuse_date + ABUSE_DELTA < date.today():
                request.session['abuse_date'] = date.today() - ABUSE_DELTA

            request.session['abuse_date'] += oneday
        return abuse_reported

    def process_request(self, request):
        request.can_report_abuse = self._can_report_abuse(request)
        request.abuse_reported = self._abuse_reported(request)
