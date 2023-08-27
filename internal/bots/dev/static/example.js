const {createApp} = Vue

createApp({
    data() {
        return {
            message: 'App page',
            form: {
                text: "",
                select: "",
                radio: ""
            }
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
        },
        submit(e) {
            let formSchema = joi.object().keys({
                text: joi.string().required(),
                select: joi.string().required(),
                radio: joi.string().required(),
            })
            const result = formSchema.validate(this.form)
            console.log("validate", result)
            if (result.error == null) {
                UIkit.notification({message: "form validate pass", status: "success"})
            } else {
                UIkit.notification({message: result.error.message, status: 'danger'})
            }
        }
    }
}).mount('#app')
