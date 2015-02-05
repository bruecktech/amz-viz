(function(){
  var app = angular.module('viz', [ ], function($interpolateProvider) {
      $interpolateProvider.startSymbol('[[');
      $interpolateProvider.endSymbol(']]');
  });

  app.controller('StackController', [ '$scope', function($scope){

    $scope.stacks = [ ] 

    var conn = new WebSocket("ws://localhost:8080/stack");

    // called when a message is received from the server
    conn.onmessage = function(e){
      $scope.$apply(function(){
        $scope.stacks = angular.fromJson(e.data).Stacks;
      });
    };

	// Log errors
	conn.onerror = function (error) {
	  console.log('WebSocket Error ' + error);
	};

  }]);

})();
