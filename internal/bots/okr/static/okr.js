const {createApp} = Vue

createApp({
    data() {
        return {
            message: 'App page'
        }
    },
    mounted() {
        console.log("uid", Global.uid)
    },
    methods: {
        greet(event) {
            UIkit.notification(`UID ${Global.uid}, ${this.message}!`)
            if (event) {
                UIkit.notification(event.target.tagName)
            }
        }
    }
}).mount('#app')
