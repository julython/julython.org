describe('JULY.CommitCollection', function() {
    describe('params()', function() {
        it('creates query parameters for languages correctly', function() {
            spyOn(window, 'PushStream').andReturn({
                addChannel: function(){},
                connect: function(){}
            });
            collection = new JULY.CommitCollection([], {
                languages: ['Ruby', 'Python']
            });
            prefix = 'limit=20&offset=0';
            expect(collection.params()).toBe(prefix + '&languages=Ruby%3BPython');
        });
    });
});
