'use strict';

/* Services */


// Demonstrate how to register services
// In this case it is a simple value service.
angular.module('myApp.services', ['ngResource']).
  factory('People', function($resource) {
      return $resource('/api/v1/people/:userId', {}, {
        query: {method:'GET', params: {limit: 10}}, isArray:false
      });
  }).
  value('version', '0.1');
