"use client"

import { ColumnDef } from "@tanstack/react-table"

import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"

import { workflowState} from "../data/data"
import { DataTableColumnHeader } from "./data-table-column-header"
import { DataTableRowActions } from "./data-table-row-actions"
import {model_Workflow} from "@/client";

export const columns: ColumnDef<model_Workflow>[] = [
  {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={
          table.getIsAllPageRowsSelected() ||
          (table.getIsSomePageRowsSelected() && "indeterminate")
        }
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
        className="translate-y-[2px]"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
        className="translate-y-[2px]"
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "id",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="ID" />
    ),
    cell: ({ row }) => <div className="w-[30px]">{row.getValue("id")}</div>,
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    cell: ({ row }) => {
      return (
        <div className="flex space-x-1">
          <div className="max-w-[300px] truncate font-medium">
            <h2>{row.getValue("name")}</h2>
            <h3 className="text-xs">{row.original.describe}</h3>
          </div>
        </div>
      )
    },
  },
  {
    accessorKey: "triggers",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Triggers" />
    ),
    cell: ({ row }) => {
      return (
        <div className="flex space-x-1">
          {row.original.triggers?.map((trigger, index) => (
            <Badge key={index} variant="outline">{trigger.type}</Badge>
          ))}
        </div>
      )
    },
  },
  {
    accessorKey: "state",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => {
      const status = workflowState.find(
        (status) => status.value === row.getValue("state")
      )

      if (!status) {
        return null
      }

      return (
        <div className="flex w-[100px] items-center">
          {status.icon && (
            <status.icon className="mr-2 h-4 w-4 text-muted-foreground" />
          )}
          <span>{status.label}</span>
        </div>
      )
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id))
    },
  },
  {
    accessorKey: "running_count",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Count" />
    ),
    cell: ({row}) => {
      return (
        <div className="flex items-center">
          <span>
            Running: <Badge variant="secondary" color={"blue"}>{row.original.running_count}</Badge>
            Successful: <Badge variant="secondary">{row.original.successful_count}</Badge>
            Canceled: <Badge variant="secondary">{row.original.canceled_count}</Badge>
            Failed: <Badge variant="secondary">{row.original.failed_count}</Badge>
          </span>
        </div>
      )
    }
  },
  {
    id: "actions",
    cell: ({ row }) => <DataTableRowActions row={row} />,
  },
]
