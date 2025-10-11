import {
  Center,
  Spinner,
  Flex,
  Spacer,
  Box,
  Heading,
  Button,
  Dialog,
} from "@chakra-ui/react";
import { Badge } from "@chakra-ui/react";
import { Server, Suffrage } from "../types/servers";
import { useState } from "react";
import { getServers, leaveCluster } from "../api/servers";
import { toaster } from "./ui/toaster";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper } from "@tanstack/react-table";
import { DataTable } from "./DataTable";
import { LogOut } from "lucide-react";
// arrow-big-up-dash

import { Tooltip } from "./ui/tooltip";

const suffrageColors: Record<Suffrage, string> = {
  Voter: "teal",
  Nonvoter: "orange",
  Staging: "gray",
};

const columnHelper = createColumnHelper<Server>();

const ServerList = () => {
  interface DeleteServerPopup {
    id: string;
    isOpen: boolean;
  }
  const [isOpen, setIsOpen] = useState<DeleteServerPopup>({
    id: "",
    isOpen: false,
  });
  const { isPending, data } = useQuery({
    queryKey: ["servers"],
    queryFn: getServers,
    refetchInterval: 5000,
  });

  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: leaveCluster,
    onSuccess: () => {
      setIsOpen({ id: "", isOpen: false });
      queryClient.invalidateQueries({ queryKey: ["servers"] });
      toaster.create({
        title: "Server left the cluster.",
        description: `The server has left the cluster successfully.`,
        type: "success",
        duration: 9000,
      });
    },
  });

  const columns = [
    columnHelper.accessor("id", {
      cell: (info) => {
        const serverId = info.getValue();
        return serverId;
      },
      header: "Server ID",
    }),
    columnHelper.accessor("is_leader", {
      cell: (info) => {
        const isLeader = info.getValue();
        return (
          <Tooltip content="Indicates if the server is leader or not">
            <Badge colorPalette={isLeader ? "purple" : "cyan"}>
              {isLeader ? "leader" : "follower"}
            </Badge>
          </Tooltip>
        );
      },
      header: "Role",
    }),
    columnHelper.accessor("suffrage", {
      cell: (info) => {
        const suffrage = info.getValue();

        return (
          <Tooltip content="Determines whether the server gets a vote">
            <Badge colorPalette={suffrageColors[suffrage] ?? "gray"}>
              {suffrage}
            </Badge>
          </Tooltip>
        );
      },
      header: "Suffrage",
    }),
    columnHelper.accessor("addr", {
      cell: (info) => info.getValue(),
      header: "Raft Address",
    }),
    columnHelper.display({
      id: "actions",
      cell: (props) => {
        return (
          <Tooltip content="Leave cluster">
            <Button
              variant="subtle"
              colorPalette="red"
              size="xs"
              onClick={() =>
                setIsOpen({ id: props.row.original.id, isOpen: true })
              }
            >
              <LogOut />
            </Button>
          </Tooltip>
        );
      },
      header: "Actions",
    }),
  ];

  if (isPending) {
    return (
      <Center h="90vh" color="white" colorPalette="teal">
        <Spinner
          borderWidth="4px"
          animationDuration="0.65s"
          color="teal.500"
          colorPalette="teal"
          size="xl"
        />
      </Center>
    );
  }

  const servers: Server[] = data?.servers || [];

  return (
    <>
      <Box className="servers" p={5}>
        <Flex>
          <Heading mb={5}>Servers</Heading>
          <Spacer />
        </Flex>
        <DataTable columns={columns} data={servers} />
      </Box>

      <Dialog.Root open={isOpen.isOpen}>
        <Dialog.Backdrop />
        <Dialog.Positioner>
          <Dialog.Content>
            <Dialog.Header fontSize="lg" fontWeight="bold">
              Leave cluster?
            </Dialog.Header>

            <Dialog.Body>
              Are you sure you want this server to leave the cluster and stop
              receiving Raft messages? To join the cluster again, you can call
              an API endpoint to join the cluster or restart the server.
            </Dialog.Body>

            <Dialog.Footer>
              <Button
                variant="outline"
                onClick={() => setIsOpen({ id: "", isOpen: false })}
              >
                Cancel
              </Button>
              <Button
                colorPalette="red"
                onClick={async () => {
                  mutation.mutate({ serverId: isOpen.id });
                }}
                ml={3}
              >
                Delete
              </Button>
            </Dialog.Footer>
          </Dialog.Content>
        </Dialog.Positioner>
      </Dialog.Root>
    </>
  );
};

export default ServerList;
