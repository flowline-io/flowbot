import FlexContainer from "@/components/flex-container";
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card";
import {Textarea} from "@/components/ui/textarea";
import {Button} from "@/components/ui/button";
import * as z from "zod";
import {useForm} from "react-hook-form";
import {zodResolver} from "@hookform/resolvers/zod";
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage,} from "@/components/ui/form"
import {Client} from "@/util/client";
import {useToast} from "@/components/ui/use-toast";
import {model_WorkflowScript, model_WorkflowScriptLang} from "@/client";
import {useNavigate} from "react-router-dom";
import {useMutation} from "@tanstack/react-query";

const formSchema = z.object({
  code: z.string(),
})

export default function ScriptFormPage() {
  const navigate = useNavigate();
  const {toast} = useToast()

  const mutation = useMutation({
    mutationFn: (data: model_WorkflowScript) => {
      return Client().workflow.postWorkflowWorkflow(data)
    },
    onSuccess: (data) => {
      console.log(data)
      if (data.status == "ok") {
        navigate(-1)
      } else {
        toast({
          title: data.status,
          description: data.message,
          variant: "destructive",
        })
      }
    },
    onError: error => {
      toast({
        title: "Error",
        description: error.message,
        variant: "destructive",
      })
    }
  })

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      code: "",
    },
  })

  function onSubmit(values: z.infer<typeof formSchema>) {
    let data: model_WorkflowScript = {
      lang: model_WorkflowScriptLang.WorkflowScriptYaml,
      code: values.code
    }
    mutation.mutate(data);
  }

  return (
    <>
      <div className="items-start justify-center gap-6 rounded-lg p-8 grid grid-cols-1">
        <div className="grid col-span-1 items-start gap-6">
          <FlexContainer>
            <Card>
              <CardHeader>
                <CardTitle>Workflow Script</CardTitle>
                <CardDescription>
                  Create/Edit script
                </CardDescription>
              </CardHeader>
              <CardContent className="grid gap-6">
                <Form {...form}>
                  <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
                    <div className="grid gap-6">
                      <div className="grid gap-2">
                        <FormField
                          control={form.control}
                          name="code"
                          render={({field}) => (
                            <FormItem>
                              <FormLabel>Script</FormLabel>
                              <FormControl>
                                <Textarea placeholder="Place input yaml" {...field} rows={30} />
                              </FormControl>
                              <FormMessage/>
                            </FormItem>
                          )}
                        />
                      </div>
                    </div>
                    <div className="grid gap-6">
                      <Button type="submit">Submit</Button>
                      <Button variant="ghost"
                              onClick={() => {
                                navigate(-1);
                              }}>Cancel</Button>
                    </div>
                  </form>
                </Form>
              </CardContent>
            </Card>
          </FlexContainer>
        </div>
      </div>
    </>
  )
}
