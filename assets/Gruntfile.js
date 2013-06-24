
var deps = [
  'js/vendor/jquery-*.js',
  'js/vendor/underscore-*.js',
  'js/vendor/backbone*.js',
  'js/vendor/knockout-*.js',
  'js/vendor/knockback-*.js',
  'js/vendor/**/*.js',
  'bootstrap/js/bootstrap-tooltip.js',
  'bootstrap/js/*.js'
];

var apps = [
  'js/src/core.js',
  'js/src/**/*.js'
];

module.exports = function(grunt) {

  grunt.initConfig({
    lint: {
      files: ['grunt.js', 'js/src/**/*.js', 'js/spec/**/*.js']
    },

    // Dev-mode - just concatenate our js files
    concat: {
      app: {
        src: apps,
        dest: '../july/static/js/app.js'
      },
      vendor: {
        src: deps,
        dest: '../july/static/js/vendor.js'
      }
    },

    // Production-mode - minify app and vendor
    uglify: {
      app: {
        src: apps,
        dest: '../july/static/js/app.js'
      },
      vendor: {
        src: deps,
        dest: '../july/static/js/vendor.js'
      }
    },

    // less compilation
    less: {
      dev: {
        options: {
            paths: ["assets/bootstrap/less"]
        },
        files: {
            "../july/static/css/main.css": "less/layout.less"
        }
      },
      prod: {
        options: {
            paths: ["assets/bootstrap/less"],
            compress: true
        },
        files: {
            "../july/static/css/main.css": "less/layout.less"
        }
      }
    },

    // Watch tasks for development
    watch: {
      less: {
        files: 'less/*.less',
        tasks: 'less:dev'
      },

      js: {
        files: 'js/src/**/*.js',
        tasks: 'concat:app'
      },

      vendor: {
        files: 'js/vendor/**/*.js',
        tasks: 'concat:vendor'
      }
    },

    // Run our Jasmine tests
    jasmine: {
      julython: {
        src: apps,
        options: {
          vendor: deps,
          specs: 'js/spec/**/*Spec.js',
          junit: {
            output: 'junit/'
          }
        }
      }
    }

  });

  grunt.loadNpmTasks('grunt-contrib-jasmine');
  grunt.loadNpmTasks('grunt-contrib-less');
  grunt.loadNpmTasks('grunt-contrib-watch');
  grunt.loadNpmTasks('grunt-contrib-concat');
  grunt.loadNpmTasks('grunt-contrib-uglify');
};
