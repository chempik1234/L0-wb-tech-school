# RUN env_var.sh ORDER_SERVICE_TABLE order_service init.sql
echo "s/\${$1}/$2/g"
sed -i "s/\${$1}/$2/g" $3