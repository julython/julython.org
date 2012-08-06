'use strict';


// Declare app level module which depends on filters, and services
angular.module('myApp', ['myApp.filters', 'myApp.services', 'myApp.directives', 'ngResource']).
  config(['$routeProvider', '$locationProvider', function($routeProvider, $locationProvider) {
    $locationProvider.html5Mode(false).hashPrefix('!');
    $routeProvider.when('/people', {templateUrl: '/app/partials/people.html', controller: PeopleController});
    $routeProvider.when('/help', {templateUrl: '/app/partials/partial2', controller: MyCtrl2});
    $routeProvider.otherwise({redirectTo: '/people'});
  }]);
