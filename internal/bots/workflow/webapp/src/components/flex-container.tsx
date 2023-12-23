import {cn} from "@/lib/utils"

export default function FlexContainer({
                         className,
                         ...props
                       }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "flex items-center justify-center [&>div]:w-full",
        className
      )}
      {...props}
    />
  )
}
