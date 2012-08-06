'use strict';

/* Controllers */


function PeopleController($resource, $scope, People) {
    People.query({}, function(data){
        $scope.users = data.models;
    });
};

PeopleController.$inject = ['$resource', '$scope', 'People'];


function MyCtrl2() {
};
MyCtrl2.$inject = [];
