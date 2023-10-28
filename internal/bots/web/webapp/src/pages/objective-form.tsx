import FlexContainer from "@/components/flex-container";
import {Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle} from "@/components/ui/card";
import {Label} from "@/components/ui/label";
import {Select, SelectContent, SelectItem, SelectLabel, SelectTrigger, SelectValue} from "@/components/ui/select";
import {Input} from "@/components/ui/input";
import {Textarea} from "@/components/ui/textarea";
import {Button} from "@/components/ui/button";
import {TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs";
import {Tabs} from "@radix-ui/react-tabs";
import {Switch} from "@/components/ui/switch";
import {cn} from "@/lib/utils";
import {format} from "date-fns"
import {Popover, PopoverTrigger} from "@radix-ui/react-popover";
import {PopoverContent} from "@/components/ui/popover";
import {Calendar} from "@/components/ui/calendar";
import {CalendarIcon} from "lucide-react";
import React from "react";
import {PersonIcon} from "@radix-ui/react-icons";
import {Dialog, DialogTrigger} from "@radix-ui/react-dialog";
import {DialogContent, DialogFooter, DialogHeader, DialogTitle} from "@/components/ui/dialog";
import {SelectGroup} from "@radix-ui/react-select";
import * as z from "zod";
import {useForm} from "react-hook-form";
import {zodResolver} from "@hookform/resolvers/zod";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import {Client} from "@/util/client";
import {useToast} from "@/components/ui/use-toast";

const formSchema = z.object({
  title: z.string().min(1).max(50),
  is_plan: z.boolean(),
  plan_start: z.date(),
  plan_end: z.date(),
  memo: z.string(),
})

export default function ObjectiveFormPage() {
  const {toast} = useToast()

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      title: "",
      is_plan: false,
    },
  })

  function onSubmit(values: z.infer<typeof formSchema>) {
    console.log(values)
    Client().okr.postOkrObjective(values).then((data) => {
      console.log(data)
      if (data.status == "ok") {
        console.log("ok", data)
      } else {
        console.log("toast", data)
        toast({
          title: data.status,
          description: data.message,
          variant: "destructive",
        })
      }
    }).catch(console.error)
  }

  return (
    <>
      <div className="items-start justify-center gap-6 rounded-lg p-8 md:grid lg:grid-cols-2 xl:grid-cols-3">
        <div className="col-span-2 grid items-start gap-6 lg:col-span-1">
          <FlexContainer>
            <Card>
              <CardHeader>
                <CardTitle>编辑目标</CardTitle>
                <CardDescription>
                  创建/编辑 目标
                </CardDescription>
              </CardHeader>
              <CardContent className="grid gap-6">
                <Tabs defaultValue="base" className="w-[600px]">
                  <TabsList>
                    <TabsTrigger value="base">基本信息</TabsTrigger>
                    <TabsTrigger value="key-result" className="hidden">关键结果</TabsTrigger>
                    <TabsTrigger value="motive" className="hidden">动机 & 可行性</TabsTrigger>
                  </TabsList>
                  <TabsContent value="base">
                    <Form {...form}>
                      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
                        <div className="grid gap-6">
                          <div className="grid gap-2">
                            <FormField
                              control={form.control}
                              name="title"
                              render={({field}) => (
                                <FormItem>
                                  <FormLabel>标题</FormLabel>
                                  <FormControl>
                                    <Input placeholder="shadcn" {...field} />
                                  </FormControl>
                                  <FormMessage/>
                                </FormItem>
                              )}
                            />
                          </div>
                          <div className="grid gap-2">
                            <Label htmlFor="subject">计划</Label>
                            <div className="grid grid-cols-3 gap-4">
                              <div className="grid gap-2">
                                <FormField
                                  control={form.control}
                                  name="is_plan"
                                  render={({field}) => (
                                    <FormItem>
                                      <FormLabel>开启</FormLabel>
                                      <FormControl>
                                        <Switch checked={field.value} onCheckedChange={field.onChange}/>
                                      </FormControl>
                                      <FormMessage/>
                                    </FormItem>
                                  )}
                                />
                              </div>
                              <div className="grid gap-2">
                                <FormField
                                  control={form.control}
                                  name="plan_start"
                                  render={({field}) => (
                                    <FormItem className="flex flex-col">
                                      <FormLabel>开始日期</FormLabel>
                                      <Popover>
                                        <PopoverTrigger asChild>
                                          <FormControl>
                                            <Button
                                              variant={"outline"}
                                              className={cn(
                                                "w-[180px] pl-3 text-left font-normal",
                                                !field.value && "text-muted-foreground"
                                              )}
                                            >
                                              {field.value ? (
                                                format(field.value, "PPP")
                                              ) : (
                                                <span>Pick a date</span>
                                              )}
                                              <CalendarIcon className="ml-auto h-4 w-4 opacity-50"/>
                                            </Button>
                                          </FormControl>
                                        </PopoverTrigger>
                                        <PopoverContent className="w-auto p-0" align="start">
                                          <Calendar
                                            mode="single"
                                            selected={field.value}
                                            onSelect={field.onChange}
                                            disabled={(date) =>
                                              date > new Date() || date < new Date("1900-01-01")
                                            }
                                            initialFocus
                                          />
                                        </PopoverContent>
                                      </Popover>
                                      <FormMessage/>
                                    </FormItem>
                                  )}
                                />
                              </div>
                              <div className="grid gap-2">
                                <FormField
                                  control={form.control}
                                  name="plan_end"
                                  render={({field}) => (
                                    <FormItem className="flex flex-col">
                                      <FormLabel>结束日期</FormLabel>
                                      <Popover>
                                        <PopoverTrigger asChild>
                                          <FormControl>
                                            <Button
                                              variant={"outline"}
                                              className={cn(
                                                "w-[180px] pl-3 text-left font-normal",
                                                !field.value && "text-muted-foreground"
                                              )}
                                            >
                                              {field.value ? (
                                                format(field.value, "PPP")
                                              ) : (
                                                <span>Pick a date</span>
                                              )}
                                              <CalendarIcon className="ml-auto h-4 w-4 opacity-50"/>
                                            </Button>
                                          </FormControl>
                                        </PopoverTrigger>
                                        <PopoverContent className="w-auto p-0" align="start">
                                          <Calendar
                                            mode="single"
                                            selected={field.value}
                                            onSelect={field.onChange}
                                            disabled={(date) =>
                                              date > new Date() || date < new Date("1900-01-01")
                                            }
                                            initialFocus
                                          />
                                        </PopoverContent>
                                      </Popover>
                                      <FormMessage/>
                                    </FormItem>
                                  )}
                                />
                              </div>
                            </div>
                          </div>
                          <div className="grid gap-2">
                            <FormField
                              control={form.control}
                              name="memo"
                              render={({field}) => (
                                <FormItem>
                                  <FormLabel>备忘</FormLabel>
                                  <FormControl>
                                    <Textarea placeholder="shadcn" {...field} />
                                  </FormControl>
                                  <FormMessage/>
                                </FormItem>
                              )}
                            />
                          </div>
                        </div>
                        <div className="grid gap-6">
                          <Button type="submit">Submit</Button>
                          <Button variant="ghost">Cancel</Button>
                        </div>
                      </form>
                    </Form>
                  </TabsContent>
                  <TabsContent value="key-result">
                    <div className="grid gap-6">
                      <div className="grid grid-cols-1 gap-4">
                        <div
                          className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                          <PersonIcon className="mt-px h-5 w-5"/>
                          <div className="space-y-1">
                            <div className="text-sm font-medium leading-none">关键结果 1</div>
                            <div className="text-sm text-muted-foreground">
                              0 -- 100
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                    <div className="grid gap-6">
                      <Dialog>
                        <DialogTrigger asChild>
                          <Button variant="outline">新建关键结果</Button>
                        </DialogTrigger>
                        <DialogContent className="sm:max-w-[425px]">
                          <DialogHeader>
                            <DialogTitle>编辑关键结果</DialogTitle>
                          </DialogHeader>
                          <div className="grid gap-4 py-4">
                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="name" className="text-right">
                                标题
                              </Label>
                              <Input
                                id="name"
                                defaultValue="Pedro Duarte"
                                className="col-span-3"
                              />
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="name" className="text-right">
                                初始值
                              </Label>
                              <Input
                                id="name"
                                defaultValue="Pedro Duarte"
                                className="col-span-3"
                                type="number"
                                value={0}
                              />
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="name" className="text-right">
                                目标值
                              </Label>
                              <Input
                                id="name"
                                defaultValue="Pedro Duarte"
                                className="col-span-3"
                                type="number"
                                value={100}
                              />
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="name" className="text-right">
                                取值方式
                              </Label>
                              <Select>
                                <SelectTrigger className="w-[180px]">
                                  <SelectValue placeholder="Select a fruit"/>
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectGroup>
                                    <SelectLabel>取值方式</SelectLabel>
                                    <SelectItem value="sum">求和</SelectItem>
                                    <SelectItem value="last">最终值</SelectItem>
                                    <SelectItem value="avg">平均值</SelectItem>
                                    <SelectItem value="max">最大值</SelectItem>
                                  </SelectGroup>
                                </SelectContent>
                              </Select>
                            </div>
                            <div className="grid grid-cols-1 gap-4">
                              <Popover>
                                <PopoverTrigger>取值方式说明</PopoverTrigger>
                                <PopoverContent>
                                  <div>
                                    - 求和：关键结果的当前值为所有记录值的和 <br/>
                                    - 最终值：关键结果的当前值为所有记录中最后记录的值 <br/>
                                    - 平均值：关键结果的当前值为所有记录中值的平均值 <br/>
                                    - 最大值：关键结果的当前值为所有记录中的最大值
                                  </div>
                                </PopoverContent>
                              </Popover>
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="name" className="text-right">
                                备忘
                              </Label>
                              <Textarea
                                className="col-span-3"
                              />
                            </div>
                          </div>
                          <DialogFooter>
                            <Button type="submit">完成</Button>
                          </DialogFooter>
                        </DialogContent>
                      </Dialog>
                    </div>
                  </TabsContent>
                  <TabsContent value="motive">
                    <div className="grid gap-6">
                      <div className="grid grid-cols-1 gap-4">
                        <div
                          className="-mx-2 flex items-start space-x-4 rounded-md p-2 transition-all hover:bg-accent hover:text-accent-foreground">
                          <PersonIcon className="mt-px h-5 w-5"/>
                          <div className="space-y-1">
                            <div className="text-sm font-medium leading-none">动机 1</div>
                            <div className="text-sm text-muted-foreground">
                              &nbsp;
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                    <div className="grid gap-6">
                      <Dialog>
                        <DialogTrigger asChild>
                          <Button variant="outline">新建动机 & 可行性</Button>
                        </DialogTrigger>
                        <DialogContent className="sm:max-w-[425px]">
                          <DialogHeader>
                            <DialogTitle>编辑动机 & 可行性</DialogTitle>
                          </DialogHeader>
                          <div className="grid gap-4 py-4">
                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="name" className="text-right">
                                类型
                              </Label>
                              <Select>
                                <SelectTrigger className="w-[180px]">
                                  <SelectValue placeholder="Select a fruit"/>
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectGroup>
                                    <SelectItem value="cat1">动机</SelectItem>
                                    <SelectItem value="cat2">可行性</SelectItem>
                                  </SelectGroup>
                                </SelectContent>
                              </Select>
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="name" className="text-right">
                                标题
                              </Label>
                              <Input
                                id="name"
                                defaultValue="Pedro Duarte"
                                className="col-span-3"
                              />
                            </div>
                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="name" className="text-right">
                                备忘
                              </Label>
                              <Textarea
                                className="col-span-3"
                              />
                            </div>
                          </div>
                          <DialogFooter>
                            <Button type="submit">完成</Button>
                          </DialogFooter>
                        </DialogContent>
                      </Dialog>
                    </div>
                  </TabsContent>
                </Tabs>
              </CardContent>
            </Card>
          </FlexContainer>
        </div>
      </div>
    </>
  )
}
