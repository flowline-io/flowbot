import { z } from "zod"
import { columns } from "./components/columns"
import { DataTable } from "./components/data-table"
import {taskSchema} from "./data/schema"
import data from "./data/tasks.json"
import {Button} from "@/components/ui/button.tsx";


// Simulate a database read for tasks.
function getTasks() {
  return z.array(taskSchema).parse(data)
}

export default function TaskPage() {
  let tasks = getTasks()

  return (
    <>
      <div className="hidden h-full flex-1 flex-col space-y-8 p-8 md:flex">
        <div className="flex items-center justify-between space-y-2">
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Workflow</h2>
            <p className="text-muted-foreground">
              Here&apos;s a list of your workflows
            </p>
          </div>
          <div className="flex items-center space-x-2">
            <Button>Create workflow</Button>
          </div>
        </div>
        <DataTable data={tasks} columns={columns} />
      </div>
    </>
  )
}
