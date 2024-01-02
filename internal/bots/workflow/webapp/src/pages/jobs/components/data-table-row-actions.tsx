"use client"

import {DotsHorizontalIcon} from "@radix-ui/react-icons"
import {Row} from "@tanstack/react-table"

import {Button} from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {useMutation} from "@tanstack/react-query";
import {Client} from "@/util/client.ts";
import {toast} from "@/components/ui/use-toast.ts";

interface DataTableRowActionsProps<TData> {
  row: Row<TData>
}

export function DataTableRowActions<TData>({row}: DataTableRowActionsProps<TData>) {

  const jobRerun = useMutation({
    mutationFn: (id: number) => {
      return Client().workflow.postWorkflowJobRerun(id)
    },
    onSuccess: (data) => {
      console.log(data)
      if (data.status == "ok") {
        alert('rerun')
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

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="flex h-8 w-8 p-0 data-[state=open]:bg-muted"
        >
          <DotsHorizontalIcon className="h-4 w-4"/>
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[160px]">
        <DropdownMenuItem onClick={() => {
          if (confirm("rerun job?")) {
            jobRerun.mutate(row.original.id)
          }
        }}>Rerun</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
