ELEMENT.locale(ELEMENT.lang.en)

var subnets = []
var applications = []
var services = []

var app = new Vue({
    el: '#app',
    data: {
        menuIndex: 'manager',
        subnets: subnets,
        applications: applications,
        services: services
    },
    created: function () {
        axios.get('/subnets').then(function (response) {
            this.subnets.push(...response.data.items)
        })
        axios.get('/applications').then(function (response) {
            this.applications.push(...response.data.items)
        })
        axios.get('/services').then(function (response) {
            this.services.push(...response.data.items)
        })
    },
    methods: {
        menuSelect: function (key, keyPath) {
            this.menuIndex = key;
        },
        updateSubnet: function (idx, state) {
            var sn = this.subnets[idx]
            var data = {
                state: state,
                rate: sn.rate,
                clients: sn.clients
            }
            axios.put('/subnets/' + idx, data).then(function (response) {
                this.subnets.length = 0;
                this.subnets.push(...response.data.items);
            })
        },
        query: function (idx) {
            var application = this.applications[idx]
            var data = {
                attempts: application.attempts,
                qname: application.qname,
                qtype: application.qtype,
                transport: application.transport
            }
            axios.put('/query/' + idx, data).then(function (response) {
                this.applications.length = 0;
                this.applications.push(...response.data.items);
            })
        },
        perf: function (idx, state) {
            var application = this.applications[idx]
            var data = {
                state: state,
                attempts: application.attempts,
                qname: application.qname,
                qtype: application.qtype,
                transport: application.transport
            }
            axios.put('/perf/' + idx, data).then(function (response) {
                this.applications.length = 0;
                this.applications.push(...response.data.items);
            })
        },
        updateService: function (idx, operation) {
            var sn = this.subnets[idx]
            var data = { operation: operation }
            axios.put('/services/' + idx, data).then(function (response) {
                this.services.length = 0;
                this.services.push(...response.data.items);
            })
        }
    }
})