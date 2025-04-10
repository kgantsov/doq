import * as React from "react";
import { Table, Flex, Badge } from "@chakra-ui/react";
import { ArrowUp01, ArrowDown01, ArrowUpAZ, ArrowDownAZ } from "lucide-react";
import {
  useReactTable,
  flexRender,
  getCoreRowModel,
  ColumnDef,
  SortingState,
  getSortedRowModel,
} from "@tanstack/react-table";

export type DataTableProps<Data extends object> = {
  data: Data[];
  columns: ColumnDef<Data, any>[];
};

export function DataTable<Data extends object>({
  data,
  columns,
}: DataTableProps<Data>) {
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const table = useReactTable({
    columns,
    data,
    getCoreRowModel: getCoreRowModel(),
    onSortingChange: setSorting,
    getSortedRowModel: getSortedRowModel(),
    state: {
      sorting,
    },
  });

  return (
    <Table.Root>
      <Table.Header>
        {table.getHeaderGroups().map((headerGroup) => (
          <Table.Row key={headerGroup.id}>
            {headerGroup.headers.map((header) => {
              // see https://tanstack.com/table/v8/docs/api/core/column-def#meta to type this correctly
              const meta: any = header.column.columnDef.meta;
              return (
                <Table.ColumnHeader
                  key={header.id}
                  onClick={header.column.getToggleSortingHandler()}
                >
                  <Flex>
                    {flexRender(
                      header.column.columnDef.header,
                      header.getContext()
                    )}
                    &nbsp;
                    {header.column.getIsSorted() ? (
                      header.column.getIsSorted() === "desc" ? (
                        <Badge>
                          {meta?.isNumeric ? (
                            <ArrowUp01
                              size={14}
                              aria-label="sorted descending"
                            />
                          ) : (
                            <ArrowUpAZ
                              size={14}
                              aria-label="sorted descending"
                            />
                          )}
                        </Badge>
                      ) : (
                        <Badge>
                          {meta?.isNumeric ? (
                            <ArrowDown01
                              size={14}
                              aria-label="sorted ascending"
                            />
                          ) : (
                            <ArrowDownAZ
                              size={14}
                              aria-label="sorted ascending"
                            />
                          )}
                        </Badge>
                      )
                    ) : null}
                  </Flex>
                </Table.ColumnHeader>
              );
            })}
          </Table.Row>
        ))}
      </Table.Header>
      <Table.Body>
        {table.getRowModel().rows.map((row) => (
          <Table.Row key={row.id}>
            {row.getVisibleCells().map((cell) => {
              return (
                <Table.Cell key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </Table.Cell>
              );
            })}
          </Table.Row>
        ))}
      </Table.Body>
    </Table.Root>
  );
}
